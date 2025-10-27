package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

func TestTryAddTask_PreStart_NoBuffer_InvalidState(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](ctx)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	ok, err := w.TryAddTask(workers.TaskValue[int](func(context.Context) int { return 1 }))
	if ok || !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected (false, ErrInvalidState), got (%v, %v)", ok, err)
	}
}

func TestTryAddTask_PreStart_Buffered_AcceptsThenRejectsWhenFull(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](ctx, workers.WithTasksBuffer(2))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ok1, err := w.TryAddTask(workers.TaskValue[int](func(context.Context) int { return 10 }))
	if !ok1 || err != nil {
		t.Fatalf("first TryAddTask expected (true, nil), got (%v, %v)", ok1, err)
	}
	ok2, err := w.TryAddTask(workers.TaskValue[int](func(context.Context) int { return 20 }))
	if !ok2 || err != nil {
		t.Fatalf("second TryAddTask expected (true, nil), got (%v, %v)", ok2, err)
	}
	ok3, err := w.TryAddTask(workers.TaskValue[int](func(context.Context) int { return 30 }))
	if ok3 || err != nil {
		t.Fatalf("third TryAddTask expected (false, nil), got (%v, %v)", ok3, err)
	}

	w.Start(ctx)

	// Drain two results, none should be errors.
	recv := 0
	deadline := time.NewTimer(1 * time.Second)
	defer deadline.Stop()
	for recv < 2 {
		select {
		case <-w.GetResults():
			recv++
		case err := <-w.GetErrors():
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for 2 results; got %d", recv)
		}
	}

	w.Close()
}

func TestTryAddTask_StopOnError_InvalidState(t *testing.T) {
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

	// Trigger cancellation with an error and read it.
	if err := w.AddTask(workers.TaskError[int](func(context.Context) error { return errors.New("boom") })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	select {
	case <-w.GetErrors():
		// ok
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for outward error")
	}

	ok, err := w.TryAddTask(workers.TaskValue[int](func(context.Context) int { return 42 }))
	if ok || !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected (false, ErrInvalidState) after cancel, got (%v, %v)", ok, err)
	}

	w.Close()
}

func TestTryAddTask_AfterClose_InvalidState(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](ctx, workers.WithStartImmediately())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	w.Close()

	ok, err := w.TryAddTask(workers.TaskValue[int](func(context.Context) int { return 1 }))
	if ok || !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected (false, ErrInvalidState) after Close, got (%v, %v)", ok, err)
	}
}
