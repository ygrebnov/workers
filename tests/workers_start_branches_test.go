package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

// Covers Start(ctx) path: ctx.Done branch stops dispatcher; AddTask returns ErrInvalidState.
func TestStart_ContextCancel_StopsDispatcher(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var w workers.Workers[int] // zero-value; Start should lazily init with defaults
	w.Start(ctx)

	// Cancel promptly and allow dispatcher to observe it.
	cancel()
	time.Sleep(20 * time.Millisecond)

	err := w.AddTask(workers.TaskValue[int](func(ctx context.Context) int { return 1 }))
	if !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}

	w.Close()
}

// Covers Start when StopOnError is enabled and outward errors is buffered (sync forward path).
func TestStart_StopOnError_BufferedOutward_SynchronousForward(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](
		ctx,
		workers.WithErrorsBuffer(1),      // outward has capacity
		workers.WithStopOnErrorBuffer(1), // small internal buffer
		workers.WithStopOnError(),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Delayed start to exercise Start logic directly.
	w.Start(ctx)

	// Trigger first error.
	if err := w.AddTask(workers.TaskError[int](func(ctx context.Context) error { return errors.New("boom") })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Should be forwarded promptly (synchronously) due to outward buffer.
	select {
	case <-w.GetErrors():
		// ok
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for outward error (buffered path)")
	}

	// Second task must not start after cancellation.
	started := make(chan struct{}, 1)
	err = w.AddTask(workers.TaskError[int](func(ctx context.Context) error { started <- struct{}{}; return nil }))
	if err == nil {
		select {
		case <-started:
			t.Fatalf("task started despite cancellation")
		case <-time.After(200 * time.Millisecond):
			// ok: did not start in time window
		}
	} else if !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}

	w.Close()
}

// Covers Start when StopOnError is enabled and outward errors is unbuffered (async forward via detached goroutine).
func TestStart_StopOnError_UnbufferedOutward_AsyncForward(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](
		ctx,
		workers.WithErrorsBuffer(0),      // saturated outward when no reader
		workers.WithStopOnErrorBuffer(1), // small internal buffer
		workers.WithStopOnError(),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Delayed start to exercise Start logic directly.
	w.Start(ctx)

	// Trigger error; forwarder will spawn detached goroutine to deliver when reader appears.
	if err := w.AddTask(workers.TaskError[int](func(ctx context.Context) error { return errors.New("boom") })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Give cancel-first a moment to propagate.
	time.Sleep(50 * time.Millisecond)

	// Verify further tasks don't start.
	started := make(chan struct{}, 1)
	err = w.AddTask(workers.TaskError[int](func(ctx context.Context) error { started <- struct{}{}; return nil }))
	if err == nil {
		select {
		case <-started:
			t.Fatalf("task started despite cancellation (unbuffered outward)")
		case <-time.After(200 * time.Millisecond):
			// ok
		}
	} else if !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}

	// Now drain the error; detached sender should deliver it once a reader appears.
	select {
	case <-w.GetErrors():
		// ok
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for outward error (unbuffered path)")
	}

	w.Close()
}
