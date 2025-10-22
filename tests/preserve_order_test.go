package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func TestWorkers_PreserveOrder_Basic(t *testing.T) {
	ctx := context.Background()
	w, err := workers.NewOptions[int](ctx, workers.WithDynamicPool(), workers.WithStartImmediately(), workers.WithPreserveOrder())
	require.NoError(t, err)

	n := 8
	for i := 0; i < n; i++ {
		ii := i
		// Make later indices finish earlier to exercise reordering.
		delay := time.Duration((n-1-ii)*5) * time.Millisecond
		require.NoError(t, w.AddTask(workers.TaskValue[int](func(context.Context) int {
			time.Sleep(delay)
			return ii
		})))
	}

	got := make([]int, 0, n)
	for i := 0; i < n; i++ {
		select {
		case v := <-w.GetResults():
			got = append(got, v)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for result %d", i)
		}
	}

	w.Close()

	expected := make([]int, 0, n)
	for i := 0; i < n; i++ {
		expected = append(expected, i)
	}
	require.Equal(t, expected, got)
}

func TestWorkers_PreserveOrder_SkipNoResult(t *testing.T) {
	ctx := context.Background()
	w, err := workers.NewOptions[int](ctx, workers.WithDynamicPool(), workers.WithStartImmediately(), workers.WithPreserveOrder())
	require.NoError(t, err)

	n := 10
	expected := make([]int, 0, n/2+1)
	for i := 0; i < n; i++ {
		ii := i
		if ii%2 == 0 {
			// Even indices produce a result but sleep longer the smaller the index, to shuffle completion.
			delay := time.Duration((n-1-ii)*3) * time.Millisecond
			require.NoError(t, w.AddTask(workers.TaskValue[int](func(context.Context) int {
				time.Sleep(delay)
				return ii
			})))
			expected = append(expected, ii)
		} else {
			// Odd indices produce no result; also vary timing.
			delay := time.Duration(ii*3) * time.Millisecond
			require.NoError(t, w.AddTask(workers.TaskError[int](func(ctx context.Context) error {
				time.Sleep(delay)
				return assertErr("nores")
			})))
		}
	}

	got := make([]int, 0, len(expected))
	for i := 0; i < len(expected); i++ {
		select {
		case v := <-w.GetResults():
			got = append(got, v)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for result %d", i)
		}
	}

	w.Close()

	require.Equal(t, expected, got)
}

func TestWorkers_PreserveOrder_ErrorWithSendResult_AdvancesCursor(t *testing.T) {
	ctx := context.Background()
	w, err := workers.NewOptions[int](ctx, workers.WithDynamicPool(), workers.WithStartImmediately(), workers.WithPreserveOrder())
	require.NoError(t, err)

	// Index 0: result
	require.NoError(t, w.AddTask(workers.TaskValue[int](func(context.Context) int { time.Sleep(5 * time.Millisecond); return 0 })))
	// Index 1: TaskFunc with error (SendResult=true signature)
	require.NoError(t, w.AddTask(workers.TaskFunc[int](func(context.Context) (int, error) { time.Sleep(8 * time.Millisecond); return 0, assertErr("boom") })))
	// Index 2: result
	require.NoError(t, w.AddTask(workers.TaskValue[int](func(context.Context) int { time.Sleep(6 * time.Millisecond); return 2 })))

	got := make([]int, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case v := <-w.GetResults():
			got = append(got, v)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for result %d", i)
		}
	}

	require.Equal(t, []int{0, 2}, got)
	w.Close()
}

func TestWorkers_PreserveOrder_StopOnError_ContiguousPrefixOnly(t *testing.T) {
	ctx := context.Background()
	w, err := workers.NewOptions[int](ctx,
		workers.WithDynamicPool(),
		workers.WithStartImmediately(),
		workers.WithPreserveOrder(),
		workers.WithStopOnError(),
		workers.WithStopOnErrorBuffer(1),
	)
	require.NoError(t, err)

	// Fast results for indices 0,1,2
	for i := 0; i < 3; i++ {
		ii := i
		require.NoError(t, w.AddTask(workers.TaskValue[int](func(context.Context) int { time.Sleep(2 * time.Millisecond); return ii })))
	}
	// Error at index 3 slightly later to allow 0..2 to complete first
	require.NoError(t, w.AddTask(workers.TaskFunc[int](func(context.Context) (int, error) { time.Sleep(30 * time.Millisecond); return 0, assertErr("stop") })))
	// Subsequent tasks that would be canceled
	for i := 4; i < 8; i++ {
		require.NoError(t, w.AddTask(workers.TaskValue[int](func(context.Context) int { time.Sleep(100 * time.Millisecond); return i })))
	}

	got := make([]int, 0, 3)
	deadline := time.After(2 * time.Second)
	for len(got) < 3 {
		select {
		case v := <-w.GetResults():
			got = append(got, v)
		case <-deadline:
			t.Fatal("timeout waiting for prefix results")
		}
	}

	require.Equal(t, []int{0, 1, 2}, got)
	w.Close()
}

func TestWorkers_PreserveOrder_ContextCancel_CleansUp_NoDeadlock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	w, err := workers.NewOptions[int](ctx, workers.WithDynamicPool(), workers.WithStartImmediately(), workers.WithPreserveOrder())
	require.NoError(t, err)

	for i := 0; i < 50; i++ {
		require.NoError(t, w.AddTask(workers.TaskValue[int](func(context.Context) int { time.Sleep(300 * time.Millisecond); return 1 })))
	}

	done := make(chan struct{})
	go func() {
		w.Close()
		close(done)
	}()

	select {
	case <-done:
		// ok, closed without deadlock
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return in time after context cancel")
	}
}

func TestWorkers_PreserveOrder_Close_CleansUp_NoDeadlock(t *testing.T) {
	ctx := context.Background()
	w, err := workers.NewOptions[int](ctx, workers.WithDynamicPool(), workers.WithStartImmediately(), workers.WithPreserveOrder())
	require.NoError(t, err)

	for i := 0; i < 50; i++ {
		require.NoError(t, w.AddTask(workers.TaskValue[int](func(context.Context) int { time.Sleep(300 * time.Millisecond); return 1 })))
	}

	done := make(chan struct{})
	go func() {
		// Give tasks a brief moment to start
		time.Sleep(30 * time.Millisecond)
		w.Close()
		close(done)
	}()

	select {
	case <-done:
		// ok, closed without deadlock
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return in time")
	}
}

func TestRunAll_PreserveOrder_Basic(t *testing.T) {
	ctx := context.Background()
	n := 12
	tasks := make([]workers.Task[int], 0, n)
	expected := make([]int, 0, n)
	for i := 0; i < n; i++ {
		ii := i
		delay := time.Duration((n-1-ii)*2) * time.Millisecond
		tasks = append(tasks, workers.TaskValue[int](func(context.Context) int { time.Sleep(delay); return ii }))
		expected = append(expected, ii)
	}

	res, err := workers.RunAll[int](ctx, tasks, workers.WithDynamicPool(), workers.WithStartImmediately(), workers.WithPreserveOrder())
	require.NoError(t, err)
	require.Equal(t, expected, res)
}
