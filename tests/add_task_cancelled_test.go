package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

// Ensures AddTask returns ErrInvalidState after cancellation triggered by StopOnError.
func TestAddTask_ReturnsInvalidState_AfterStopOnErrorCancellation(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](
		ctx,
		workers.WithErrorsBuffer(1),      // outward buffered for prompt forward
		workers.WithStopOnErrorBuffer(1), // small internal buffer
		workers.WithStopOnError(),
	)
	if err != nil {
		t.Fatalf("NewOptions returned error: %v", err)
	}

	w.Start(ctx)

	// Trigger cancellation via first error.
	if err := w.AddTask(workers.TaskError[int](func(ctx context.Context) error { return errors.New("boom") })); err != nil {
		t.Fatalf("AddTask returned error: %v", err)
	}

	// Wait for the error to be forwarded to ensure cancellation has occurred.
	select {
	case <-w.GetErrors():
		// ok
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for outward error after StopOnError")
	}

	// Now AddTask should fail with ErrInvalidState due to internal cancellation.
	err = w.AddTask(workers.TaskValue[int](func(ctx context.Context) int { return 42 }))
	if !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}

	w.Close()
}
