package tests

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func TestRunAll_HappyPath(t *testing.T) {
	ctx := context.Background()
	// Build a batch of simple tasks returning i*2.
	n := 10
	tasks := make([]workers.Task[int], 0, n)
	expected := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		tasks = append(tasks, workers.TaskValue[int](func(ctx context.Context) int { return i * 2 }))
		expected = append(expected, i*2)
	}

	results, err := workers.RunAll[int](ctx, tasks, workers.WithDynamicPool())
	require.NoError(t, err)
	require.Len(t, results, n)

	// Results are completion-ordered; compare as sets by sorting.
	sort.Ints(results)
	sort.Ints(expected)
	require.Equal(t, expected, results)
}

func TestRunAll_StopOnError_CancelsRemaining(t *testing.T) {
	ctx := context.Background()

	// 1,2 finish quickly; 3 errors; 4,5 are slow and should be canceled.
	tasks := []workers.Task[int]{
		workers.TaskValue[int](func(ctx context.Context) int { time.Sleep(10 * time.Millisecond); return 1 }),
		workers.TaskValue[int](func(ctx context.Context) int { time.Sleep(15 * time.Millisecond); return 2 }),
		workers.TaskError[int](func(ctx context.Context) error { return assertErr("boom") }),
		workers.TaskValue[int](func(ctx context.Context) int { time.Sleep(time.Second); return 4 }),
		workers.TaskValue[int](func(ctx context.Context) int { time.Sleep(time.Second); return 5 }),
	}

	results, err := workers.RunAll[int](
		ctx,
		tasks,
		workers.WithFixedPool(uint(len(tasks))),
		workers.WithStopOnError(),
		workers.WithStopOnErrorBuffer(2),
	)

	// Expect an error (first error triggers cancellation).
	require.Error(t, err)

	// At least the first two quick tasks likely completed before cancellation.
	// We don't assert exact count due to scheduling, but results should be <= 2.
	require.LessOrEqual(t, len(results), 2)
	for _, r := range results {
		require.Contains(t, []int{1, 2}, r)
	}
}

func TestRunAll_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	tasks := []workers.Task[int]{
		workers.TaskValue[int](func(ctx context.Context) int { time.Sleep(150 * time.Millisecond); return 1 }),
		workers.TaskValue[int](func(ctx context.Context) int { time.Sleep(150 * time.Millisecond); return 2 }),
	}

	results, err := workers.RunAll[int](ctx, tasks, workers.WithDynamicPool())
	// Expect context cancellation to surface through task errors aggregation.
	require.Error(t, err)
	require.Empty(t, results)
}

// assertErr is a tiny helper that returns a sentinel error matching the tests intent.
// We don't need a typed error; a consistent string is sufficient for behavior checks.
func assertErr(msg string) error { return &stringError{s: msg} }

type stringError struct{ s string }

func (e *stringError) Error() string { return e.s }
