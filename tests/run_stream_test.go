package tests

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

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
	if err != nil {
		t.Fatalf("RunStream setup error: %v", err)
	}

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
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(errors))
	}
	if len(results) != n {
		t.Fatalf("expected %d results, got %d", n, len(results))
	}
	sort.Ints(results)
	if !reflect.DeepEqual(expected, results) {
		t.Fatalf("unexpected results: got=%v want=%v", results, expected)
	}
}

func TestRunStream_StopOnError_StopsForwardingAndCloses(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int], 8)

	out, errs, err := workers.RunStream[int](ctx, in, workers.WithDynamicPool(), workers.WithStopOnError())
	if err != nil {
		t.Fatalf("RunStream setup error: %v", err)
	}

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
	if len(results) > 1 {
		t.Fatalf("expected at most 1 result, got %d", len(results))
	}
	if len(results) == 1 && results[0] != 1 {
		t.Fatalf("unexpected result value, got=%d want=1", results[0])
	}
	if len(errors) != 1 {
		t.Fatalf("expected exactly 1 outward error, got %d", len(errors))
	}
	if errors[0] == nil {
		t.Fatalf("expected non-nil error")
	}
}

func TestRunStream_PreserveOrder(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int], 8)

	out, errs, err := workers.RunStream[int](ctx, in, workers.WithDynamicPool(), workers.WithPreserveOrder())
	if err != nil {
		t.Fatalf("RunStream setup error: %v", err)
	}

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

	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(errors))
	}
	if !reflect.DeepEqual(expected, results) {
		t.Fatalf("unexpected ordered results: got=%v want=%v", results, expected)
	}
}

func TestRunStream_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	in := make(chan workers.Task[int], 8)

	out, errs, err := workers.RunStream[int](ctx, in, workers.WithDynamicPool())
	if err != nil {
		t.Fatalf("RunStream setup error: %v", err)
	}

	// Enqueue a few long tasks; context deadline should cancel them.
	for i := 0; i < 4; i++ {
		in <- workers.TaskValue[int](func(context.Context) int { time.Sleep(200 * time.Millisecond); return 7 })
	}
	close(in)

	results := collectResultsWithTimeout[int](t, out, 2*time.Second)
	errors := collectErrorsWithTimeout(t, errs, 2*time.Second)

	// Expect cancellations to produce errors; no successful results required.
	if len(results) != 0 {
		t.Fatalf("expected no results, got %d", len(results))
	}
	if len(errors) == 0 {
		t.Fatalf("expected some errors due to cancellation, got 0")
	}
}

func TestRunStream_ClosesChannels_OnInputClose(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int], 8)

	out, errs, err := workers.RunStream[int](ctx, in, workers.WithDynamicPool())
	if err != nil {
		t.Fatalf("RunStream setup error: %v", err)
	}

	for i := 0; i < 5; i++ {
		v := i
		in <- workers.TaskValue[int](func(context.Context) int { return v })
	}
	close(in)

	// After input closes and tasks finish, both channels should be closed within timeout.
	results := collectResultsWithTimeout[int](t, out, 2*time.Second)
	errors := collectErrorsWithTimeout(t, errs, 2*time.Second)

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(errors))
	}
}
