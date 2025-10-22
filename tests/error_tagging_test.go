package tests

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func TestErrorTagging_Disabled_NoMeta(t *testing.T) {
	ctx := context.Background()
	w, err := workers.NewOptions[string](ctx, workers.WithStartImmediately(), workers.WithDynamicPool())
	require.NoError(t, err)

	n := 5
	msgs := make([]string, n)
	for i := 0; i < n; i++ {
		msg := fmt.Sprintf("e%d", i)
		msgs[i] = msg
		require.NoError(t, w.AddTask(workers.TaskError[string](func(context.Context) error { return fmt.Errorf(msg) })))
	}

	received := make([]string, 0, n)
	for i := 0; i < n; i++ {
		select {
		case e := <-w.GetErrors():
			received = append(received, e.Error())
			id, ok := workers.ExtractTaskID(e)
			require.False(t, ok, "unexpected task id present: %v", id)
			_, ok = workers.ExtractTaskIndex(e)
			require.False(t, ok, "unexpected task index present")
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for error %d", i)
		}
	}

	sort.Strings(received)
	sort.Strings(msgs)
	require.Equal(t, msgs, received)

	w.Close()
}

func TestErrorTagging_Enabled_AssignsIDAndIndex(t *testing.T) {
	ctx := context.Background()
	w, err := workers.NewOptions[string](ctx, workers.WithStartImmediately(), workers.WithDynamicPool(), workers.WithErrorTagging())
	require.NoError(t, err)

	n := 5
	ids := make([]any, n)
	for i := 0; i < n; i++ {
		ids[i] = fmt.Sprintf("id-%d", i)
		require.NoError(t, w.AddTask(workers.TaskErrorWithID[string](ids[i], func(context.Context) error { return fmt.Errorf("boom-%d", i) })))
	}

	recv := 0
	seen := map[int]any{}
	for recv < n {
		select {
		case e := <-w.GetErrors():
			id, ok := workers.ExtractTaskID(e)
			require.True(t, ok)
			idx, ok := workers.ExtractTaskIndex(e)
			require.True(t, ok)
			seen[idx] = id
			recv++
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for error %d", recv)
		}
	}

	for i := 0; i < n; i++ {
		require.Equal(t, ids[i], seen[i], "id mismatch for index %d", i)
	}

	w.Close()
}

func TestErrorTagging_StopOnError_FirstTagged(t *testing.T) {
	ctx := context.Background()
	w, err := workers.NewOptions[string](
		ctx,
		workers.WithStartImmediately(),
		workers.WithDynamicPool(),
		workers.WithErrorTagging(),
		workers.WithStopOnError(),
	)
	require.NoError(t, err)

	firstID := "first"
	secondID := "second"

	require.NoError(t, w.AddTask(workers.TaskErrorWithID[string](firstID, func(context.Context) error { return fmt.Errorf("first-err") })))
	// Second AddTask may fail if cancellation is already triggered; tolerate ErrInvalidState.
	_ = w.AddTask(workers.TaskErrorWithID[string](secondID, func(context.Context) error { return fmt.Errorf("second-err") }))

	select {
	case e := <-w.GetErrors():
		id, ok := workers.ExtractTaskID(e)
		require.True(t, ok)
		// Either firstID or secondID may error first, depending on scheduling.
		require.Contains(t, []any{any(firstID), any(secondID)}, id)
		idx, ok := workers.ExtractTaskIndex(e)
		require.True(t, ok)
		require.Contains(t, []int{0, 1}, idx)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first error")
	}

	w.Close()
}

func TestErrorTagging_Panic_Tagged(t *testing.T) {
	ctx := context.Background()
	w, err := workers.NewOptions[int](ctx, workers.WithStartImmediately(), workers.WithDynamicPool(), workers.WithErrorTagging())
	require.NoError(t, err)

	panicID := "panic-task"
	require.NoError(t, w.AddTask(workers.TaskValueWithID[int](panicID, func(context.Context) int { panic("boom") })))

	select {
	case e := <-w.GetErrors():
		require.True(t, errors.Is(e, workers.ErrTaskPanicked))
		id, ok := workers.ExtractTaskID(e)
		require.True(t, ok)
		require.Equal(t, any(panicID), id)
		idx, ok := workers.ExtractTaskIndex(e)
		require.True(t, ok)
		require.Equal(t, 0, idx)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for panic error")
	}

	w.Close()
}

func TestErrorTagging_Cancel_Tagged(t *testing.T) {
	outerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w, err := workers.NewOptions[string](outerCtx, workers.WithStartImmediately(), workers.WithDynamicPool(), workers.WithErrorTagging())
	require.NoError(t, err)

	cancelID := "cancel-task"
	started := make(chan struct{}, 1)
	// Task waits for ctx.Done then returns a concrete error, and signals when started.
	blocker := workers.TaskErrorWithID[string](cancelID, func(ctx context.Context) error {
		select {
		case started <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return fmt.Errorf("canceled: %v", ctx.Err())
	})

	require.NoError(t, w.AddTask(blocker))

	// Ensure the task started before cancel.
	select {
	case <-started:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("task did not start in time")
	}

	// Cancel the outer context to trigger internal cancellation without closing channels yet.
	cancel()

	select {
	case e := <-w.GetErrors():
		id, ok := workers.ExtractTaskID(e)
		require.True(t, ok)
		require.Equal(t, any(cancelID), id)
		idx, ok := workers.ExtractTaskIndex(e)
		require.True(t, ok)
		require.Equal(t, 0, idx)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cancel error")
	}

	w.Close()
}
