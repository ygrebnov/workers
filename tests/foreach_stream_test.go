package tests

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

func TestForEachStream_HappyPath(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 16)

	errs, err := workers.ForEachStream[int](ctx, in, func(ctx context.Context, v int) error { return nil })
	if err != nil {
		t.Fatalf("ForEachStream setup error: %v", err)
	}

	// produce
	n := 20
	for i := 0; i < n; i++ {
		in <- i
	}
	close(in)

	// drain errors
	var gotErrs []error
	deadline := time.After(2 * time.Second)
loop:
	for {
		select {
		case e, ok := <-errs:
			if !ok {
				break loop
			}
			gotErrs = append(gotErrs, e)
		case <-deadline:
			t.Fatalf("timeout waiting for errs close, collected=%d", len(gotErrs))
		}
	}
	if len(gotErrs) != 0 {
		t.Fatalf("expected no errors, got %d", len(gotErrs))
	}
}

func TestForEachStream_StopOnError_CancelsRemaining(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 16)

	// Use small fixed pool so at most a couple tasks start before error triggers cancellation.
	errs, err := workers.ForEachStream[int](ctx, in, func(ctx context.Context, v int) error {
		if v == 1 {
			return assertErr("boom")
		}
		// small delay to let two tasks overlap
		time.Sleep(20 * time.Millisecond)
		return nil
	}, workers.WithFixedPool(2), workers.WithStopOnError())
	if err != nil {
		t.Fatalf("ForEachStream setup error: %v", err)
	}

	// send a handful; close input
	for i := 0; i < 10; i++ {
		in <- i
	}
	close(in)

	// collect outward errors (StopOnError should forward exactly one)
	var gotErrs []error
	deadline := time.After(2 * time.Second)
loop:
	for {
		select {
		case e, ok := <-errs:
			if !ok {
				break loop
			}
			gotErrs = append(gotErrs, e)
		case <-deadline:
			t.Fatalf("timeout waiting for errs close, collected=%d", len(gotErrs))
		}
	}
	if len(gotErrs) != 1 {
		t.Fatalf("expected exactly 1 outward error, got %d", len(gotErrs))
	}
}

func TestForEachStream_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	in := make(chan int, 8)

	errs, err := workers.ForEachStream[int](ctx, in, func(ctx context.Context, v int) error {
		time.Sleep(150 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachStream setup error: %v", err)
	}

	for i := 0; i < 4; i++ {
		in <- i
	}
	close(in)

	// Expect cancellations to surface as errors; drain with timeout.
	var gotErrs []error
	deadline := time.After(2 * time.Second)
loop:
	for {
		select {
		case e, ok := <-errs:
			if !ok {
				break loop
			}
			gotErrs = append(gotErrs, e)
		case <-deadline:
			t.Fatalf("timeout waiting for errs close, collected=%d", len(gotErrs))
		}
	}
	if len(gotErrs) == 0 {
		t.Fatalf("expected some cancellation errors, got 0")
	}
}

func TestForEachStream_ClosesErrors_OnInputClose(t *testing.T) {
	ctx := context.Background()
	in := make(chan int, 8)

	processed := make(chan int, 16)
	var mu sync.Mutex
	seen := make([]int, 0, 8)

	errs, err := workers.ForEachStream[int](ctx, in, func(ctx context.Context, v int) error {
		processed <- v
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachStream setup error: %v", err)
	}

	for i := 0; i < 8; i++ {
		in <- i
	}
	close(in)

	// Wait for errs to close and also gather processed values via processed channel (non-blocking with timeout)
	done := make(chan struct{})
	go func() {
		for v := range processed {
			mu.Lock()
			seen = append(seen, v)
			mu.Unlock()
		}
		close(done)
	}()

	deadline := time.After(5 * time.Second)
loop:
	for {
		select {
		case _, ok := <-errs:
			if !ok {
				break loop
			}
			// ignore values; we expect none in happy path
		case <-deadline:
			t.Fatalf("timeout waiting for errs close")
		}
	}
	// processed channel will be left open; close it now since ForEachStream does not own it.
	close(processed)
	<-done

	if len(seen) != 8 {
		// allow order differences only
		sort.Ints(seen)
		expected := make([]int, 8)
		for i := 0; i < 8; i++ {
			expected[i] = i
		}
		if len(seen) != len(expected) {
			t.Fatalf("expected %d processed, got %d", len(expected), len(seen))
		}
		for i := range expected {
			if seen[i] != expected[i] {
				t.Fatalf("unexpected processed set: got=%v expected=%v", seen, expected)
			}
		}
	}
}
