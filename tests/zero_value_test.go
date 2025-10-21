package tests

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func TestZeroValue_AddTaskBeforeStart_ReturnsInvalidState(t *testing.T) {
	var w workers.Workers[int] // zero-value

	err := w.AddTask(workers.TaskValue[int](func(ctx context.Context) int { return 1 }))
	require.ErrorIs(t, err, workers.ErrInvalidState)
}

func TestZeroValue_Start_InitializesDefaults_AndRunsTasks(t *testing.T) {
	ctx := context.Background()
	var w workers.Workers[int] // zero-value

	w.Start(ctx)

	// Enqueue several tasks and then Close; Close waits for in-flight tasks and closes channels.
	const n = 5
	for i := 0; i < n; i++ {
		x := i // capture loop variable
		require.NoError(t, w.AddTask(workers.TaskValue[int](func(ctx context.Context) int { return x * x })))
	}

	w.Close()

	// Drain channels and verify results; errors should be empty.
	results := make([]int, 0, n)
	errs := 0
	resultsClosed, errorsClosed := false, false
	for !(resultsClosed && errorsClosed) {
		select {
		case r, ok := <-w.GetResults():
			if !ok {
				resultsClosed = true
				continue
			}
			results = append(results, r)
		case _, ok := <-w.GetErrors():
			if !ok {
				errorsClosed = true
				continue
			}
			errs++
		}
	}

	require.Equal(t, 0, errs)
	require.Len(t, results, n)
	sort.Ints(results)
	expected := []int{0, 1, 4, 9, 16}
	require.Equal(t, expected, results)
}
