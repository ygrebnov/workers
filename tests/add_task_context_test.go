package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

// Ensures AddTaskContext returns ErrInvalidState before Start when no tasks buffer exists.
func TestAddTaskContext_PreStart_NoBuffer_ReturnsInvalidState(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](ctx /* no WithTasksBuffer */)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if err := w.AddTaskContext(callCtx, workers.TaskValue[int](func(context.Context) int { return 1 })); !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
}

// Ensures AddTaskContext times out before Start when the buffer is full and Start hasn't begun draining.
func TestAddTaskContext_PreStart_BufferFull_TimesOut(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](ctx, workers.WithTasksBuffer(2))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Fill the buffer before Start (plain AddTask blocks if buffer is full; here we fill exactly cap).
	if err := w.AddTask(workers.TaskValue[int](func(context.Context) int { return 1 })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := w.AddTask(workers.TaskValue[int](func(context.Context) int { return 2 })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Third add with short timeout should time out (no Start yet to drain the queue).
	callCtx, cancel := context.WithTimeout(ctx, 80*time.Millisecond)
	defer cancel()
	err = w.AddTaskContext(callCtx, workers.TaskValue[int](func(context.Context) int { return 3 }))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}

	// Start and drain exactly two results; the timed-out task must not have been enqueued.
	w.Start(ctx)

	received := 0
	deadline := time.NewTimer(1 * time.Second)
	defer deadline.Stop()
	for received < 2 {
		select {
		case <-w.GetResults():
			received++
		case err := <-w.GetErrors():
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for 2 results; got %d", received)
		}
	}

	w.Close()
}

// Ensures a waiting AddTaskContext before Start unblocks successfully after Start begins draining the buffer.
func TestAddTaskContext_PreStart_BufferFull_ThenStartUnblocks(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](ctx, workers.WithTasksBuffer(1))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Fill buffer with 1 element before Start.
	if err := w.AddTask(workers.TaskValue[int](func(context.Context) int { return 10 })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		callCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
		done <- w.AddTaskContext(callCtx, workers.TaskValue[int](func(context.Context) int { return 20 }))
	}()

	// Give the goroutine a chance to get blocked.
	time.Sleep(50 * time.Millisecond)
	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("AddTaskContext should be blocked before Start when buffer is full")
		}
		// If it errored, it should be timeout; but better to proceed and let Start unblock path below.
	default:
		// still blocked as expected
	}

	w.Start(ctx)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("AddTaskContext should have succeeded after Start, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for AddTaskContext to unblock after Start")
	}

	// Expect 2 results, then Close.
	want := 2
	recv := 0
	deadline := time.NewTimer(1 * time.Second)
	defer deadline.Stop()
	for recv < want {
		select {
		case <-w.GetResults():
			recv++
		case err := <-w.GetErrors():
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for %d results; got %d", want, recv)
		}
	}

	w.Close()
}

// Ensures after StopOnError cancellation, AddTaskContext returns ErrInvalidState promptly.
func TestAddTaskContext_StopOnError_CancelFast(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](
		ctx,
		workers.WithStopOnError(),
		workers.WithStopOnErrorBuffer(1),
		workers.WithErrorsBuffer(1),
		workers.WithStartImmediately(),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Trigger cancellation via first error, and read it to ensure forwarder runs.
	if err := w.AddTask(workers.TaskError[int](func(context.Context) error { return errors.New("boom") })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	select {
	case <-w.GetErrors():
		// ok
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for outward error")
	}

	// Now AddTaskContext should fail fast with ErrInvalidState, not timeout.
	callCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	err = w.AddTaskContext(callCtx, workers.TaskValue[int](func(context.Context) int { return 7 }))
	if !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState after cancel, got %v", err)
	}

	w.Close()
}

// Ensures after Close, AddTaskContext returns ErrInvalidState promptly regardless of caller timeout.
func TestAddTaskContext_AfterClose_ReturnsInvalidState(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](ctx, workers.WithStartImmediately())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	w.Close()

	callCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	err = w.AddTaskContext(callCtx, workers.TaskValue[int](func(context.Context) int { return 1 }))
	if !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState after Close, got %v", err)
	}
}
