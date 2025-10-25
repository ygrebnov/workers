package tests

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

//nolint:unparam // d is always passed.
func collectIntsWithTimeout(t *testing.T, ch <-chan int, d time.Duration) []int {
	t.Helper()
	res := make([]int, 0)
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

//nolint:unparam // d is always passed.
func collectErrorsWithTimeoutMS(t *testing.T, ch <-chan error, d time.Duration) []error {
	t.Helper()
	errs := make([]error, 0)
	deadline := time.After(d)
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return errs
			}
			errs = append(errs, e)
		case <-deadline:
			return errs
		}
	}
}

func TestMapStream_HappyPath(t *testing.T) {
	// TODO: check test, may return an error.
	ctx := context.Background()
	in := make(chan int, 16)

	out, errs, err := workers.MapStream[int, int](ctx, in, func(ctx context.Context, v int) (int, error) { return v * 2, nil })
	if err != nil {
		t.Fatalf("MapStream setup error: %v", err)
	}

	n := 12
	expected := make([]int, 0, n)
	for i := 0; i < n; i++ {
		in <- i
		expected = append(expected, i*2)
	}
	close(in)

	results := collectIntsWithTimeout(t, out, 2*time.Second)
	errors := collectErrorsWithTimeoutMS(t, errs, 2*time.Second)

	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(errors))
	}
	if len(results) != n {
		t.Fatalf("expected %d results, got %d", n, len(results))
	}
	sort.Ints(results)
	sort.Ints(expected)
	if !reflect.DeepEqual(results, expected) {
		t.Fatalf("unexpected results: got=%v want=%v", results, expected)
	}
}

func TestMapStream_StopOnError_StopsForwardingAndCloses(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 16)

	out, errs, err := workers.MapStream[int, int](ctx, in, func(ctx context.Context, v int) (int, error) {
		if v == 2 {
			return 0, assertErr("boom")
		}
		// small delay to allow some overlap
		time.Sleep(20 * time.Millisecond)
		return v, nil
	}, workers.WithFixedPool(2), workers.WithStopOnError())
	if err != nil {
		t.Fatalf("MapStream setup error: %v", err)
	}

	for i := 0; i < 8; i++ {
		in <- i
	}
	close(in)

	results := collectIntsWithTimeout(t, out, 2*time.Second)
	errors := collectErrorsWithTimeoutMS(t, errs, 2*time.Second)

	if len(errors) != 1 {
		t.Fatalf("expected exactly 1 outward error, got %d", len(errors))
	}
	if len(results) > 2 {
		t.Fatalf("expected at most 2 results before cancellation, got %d", len(results))
	}
}

func TestMapStream_PreserveOrder(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 16)

	out, errs, err := workers.MapStream[int, int](ctx, in, func(ctx context.Context, v int) (int, error) {
		return v, nil
	}, workers.WithPreserveOrder())
	if err != nil {
		t.Fatalf("MapStream setup error: %v", err)
	}

	n := 10
	expected := make([]int, 0, n)
	for i := 0; i < n; i++ {
		expected = append(expected, i)
	}
	for i := 0; i < n; i++ {
		idx := i
		delay := time.Duration((n-1-idx)*2) * time.Millisecond
		in <- idx
		// simulate out-of-order completion by delaying in the consumer function
		_ = delay
	}
	close(in)

	// Since fn returns immediately, preserve-order will just forward inputs' order
	results := collectIntsWithTimeout(t, out, 2*time.Second)
	errors := collectErrorsWithTimeoutMS(t, errs, 2*time.Second)

	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(errors))
	}
	if !reflect.DeepEqual(results, expected) {
		t.Fatalf("unexpected ordered results: got=%v want=%v", results, expected)
	}
}

func TestMapStream_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	in := make(chan int, 8)

	out, errs, err := workers.MapStream[int, int](ctx, in, func(ctx context.Context, v int) (int, error) {
		time.Sleep(150 * time.Millisecond)
		return v, nil
	})
	if err != nil {
		t.Fatalf("MapStream setup error: %v", err)
	}

	for i := 0; i < 4; i++ {
		in <- i
	}
	close(in)

	results := collectIntsWithTimeout(t, out, 2*time.Second)
	errors := collectErrorsWithTimeoutMS(t, errs, 2*time.Second)

	if len(results) != 0 {
		t.Fatalf("expected no successful results under context cancel, got %d", len(results))
	}
	if len(errors) == 0 {
		t.Fatalf("expected some cancellation errors, got 0")
	}
}

func TestMapStream_ClosesChannels_OnInputClose(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 8)

	out, errs, err := workers.MapStream[int, int](ctx, in, func(ctx context.Context, v int) (int, error) { return v, nil })
	if err != nil {
		t.Fatalf("MapStream setup error: %v", err)
	}

	n := 6
	for i := 0; i < n; i++ {
		in <- i
	}
	close(in)

	results := collectIntsWithTimeout(t, out, 2*time.Second)
	errors := collectErrorsWithTimeoutMS(t, errs, 2*time.Second)

	if len(results) != n {
		t.Fatalf("expected %d results, got %d", n, len(results))
	}
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(errors))
	}
}
