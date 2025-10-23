package tests

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

// helper: collect from a results channel until it closes or timeout expires.
func collectResultsWithTimeout[R any](t *testing.T, ch <-chan R, d time.Duration) []R {
	t.Helper()
	res := make([]R, 0)
	deadline := time.After(d)
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return res
			}
			res = append(res, v)
		case <-deadline:
			return res
		}
	}
}

// helper: collect all errors until close or timeout.
//
//nolint:unparam // d is always provided.
func collectErrorsWithTimeout(t *testing.T, ch <-chan error, d time.Duration) []error {
	t.Helper()
	errs := make([]error, 0)
	deadline := time.After(d)
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return errs
			}
			errs = append(errs, v)
		case <-deadline:
			return errs
		}
	}
}

func TestRunStream_HappyPath(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int], 8)

	out, errs, err := workers.RunStream[int](ctx, in, workers.WithDynamicPool())
	require.NoError(t, err)

	// Produce a few tasks that complete successfully.
	n := 10
	expected := make([]int, 0, n)
	for i := 0; i < n; i++ {
		v := i
		expected = append(expected, v)
		in <- workers.TaskValue[int](func(context.Context) int { return v })
	}
	close(in)

	results := collectResultsWithTimeout[int](t, out, 2*time.Second)
	errors := collectErrorsWithTimeout(t, errs, 2*time.Second)

	// No runtime errors expected.
	require.Len(t, errors, 0)
	require.Len(t, results, n)
	sort.Ints(results)
	require.Equal(t, expected, results)
}

func TestRunStream_StopOnError_StopsForwardingAndCloses(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int], 8)

	out, errs, err := workers.RunStream[int](ctx, in, workers.WithDynamicPool(), workers.WithStopOnError())
	require.NoError(t, err)

	// Send one quick success, then an immediate error, then a few slow tasks.
	in <- workers.TaskValue[int](func(context.Context) int { time.Sleep(10 * time.Millisecond); return 1 })
	in <- workers.TaskError[int](func(context.Context) error { return assertErr("boom") })
	for i := 0; i < 3; i++ {
		in <- workers.TaskValue[int](func(context.Context) int { time.Sleep(500 * time.Millisecond); return 42 })
	}
	close(in)

	results := collectResultsWithTimeout[int](t, out, 2*time.Second)
	errors := collectErrorsWithTimeout(t, errs, 2*time.Second)

	// StopOnError forwards exactly one error outward.
	require.LessOrEqual(t, len(results), 1)
	if len(results) == 1 {
		require.Equal(t, 1, results[0])
	}
	require.Equal(t, 1, len(errors))
	require.Error(t, errors[0])
}

func TestRunStream_PreserveOrder(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int], 8)

	out, errs, err := workers.RunStream[int](ctx, in, workers.WithDynamicPool(), workers.WithPreserveOrder())
	require.NoError(t, err)

	n := 8
	expected := make([]int, 0, n)
	for i := 0; i < n; i++ {
		expected = append(expected, i)
	}
	// Emit tasks with decreasing delays so completion is out-of-order; preserve-order should reorder to input sequence.
	for i := 0; i < n; i++ {
		idx := i
		delay := time.Duration((n-1-idx)*3) * time.Millisecond
		in <- workers.TaskValue[int](func(context.Context) int { time.Sleep(delay); return idx })
	}
	close(in)

	results := collectResultsWithTimeout[int](t, out, 2*time.Second)
	errors := collectErrorsWithTimeout(t, errs, 2*time.Second)

	require.Empty(t, errors)
	require.Equal(t, expected, results)
}

func TestRunStream_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	in := make(chan workers.Task[int], 8)

	out, errs, err := workers.RunStream[int](ctx, in, workers.WithDynamicPool())
	require.NoError(t, err)

	// Enqueue a few long tasks; context deadline should cancel them.
	for i := 0; i < 4; i++ {
		in <- workers.TaskValue[int](func(context.Context) int { time.Sleep(200 * time.Millisecond); return 7 })
	}
	close(in)

	results := collectResultsWithTimeout[int](t, out, 2*time.Second)
	errors := collectErrorsWithTimeout(t, errs, 2*time.Second)

	// Expect cancellations to produce errors; no successful results required.
	require.Empty(t, results)
	require.NotEmpty(t, errors)
}

func TestRunStream_ClosesChannels_OnInputClose(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int], 8)

	out, errs, err := workers.RunStream[int](ctx, in, workers.WithDynamicPool())
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		v := i
		in <- workers.TaskValue[int](func(context.Context) int { return v })
	}
	close(in)

	// After input closes and tasks finish, both channels should be closed within timeout.
	results := collectResultsWithTimeout[int](t, out, 2*time.Second)
	errors := collectErrorsWithTimeout(t, errs, 2*time.Second)

	require.Len(t, results, 5)
	require.Empty(t, errors)
}
