package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

// Ensures AddTask returns ErrInvalidState after cancellation when outward errors is unbuffered (async forward).
func TestAddTask_ReturnsInvalidState_AfterStopOnErrorCancellation_UnbufferedOutward(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](
		ctx,
		workers.WithDynamicPool(),
		workers.WithErrorsBuffer(0),      // outward unbuffered -> detached sender path
		workers.WithStopOnErrorBuffer(1), // small internal buffer
		workers.WithStopOnError(),
	)
	if err != nil {
		t.Fatalf("NewOptions failed: %v", err)
	}

	w.Start(ctx)

	// Trigger cancellation via first error.
	if err := w.AddTask(workers.TaskError[int](func(ctx context.Context) error { return errors.New("boom") })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Give forwarder a moment to receive from internal buffer and cancel before any reader is attached.
	time.Sleep(50 * time.Millisecond)

	// Now AddTask should fail with ErrInvalidState due to internal cancellation (even before error is drained).
	err = w.AddTask(workers.TaskValue[int](func(ctx context.Context) int { return 7 }))
	if !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}

	// Drain the outward error; detached sender should deliver once a reader appears.
	select {
	case <-w.GetErrors():
		// ok
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for outward error (unbuffered outward)")
	}

	w.Close()
}
