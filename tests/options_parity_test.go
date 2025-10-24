package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/ygrebnov/workers"
)

func TestDefaultsParity_New_vs_NewOptions(t *testing.T) {
	ctx := context.Background()

	w1, err := workers.NewOptions[int](ctx)
	if err != nil {
		t.Fatalf("NewOptions failed: %v", err)
	}
	w2, err := workers.NewOptions[int](ctx)
	if err != nil {
		t.Fatalf("NewOptions failed: %v", err)
	}

	// By default, StartImmediately is false and TasksBufferSize is 0.
	// Adding a task before Start should return ErrInvalidState for both.
	task := workers.TaskValue[int](func(context.Context) int { return 42 })
	if err := w1.AddTask(task); !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState for w1, got %v", err)
	}
	if err := w2.AddTask(task); !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState for w2, got %v", err)
	}

	// After Start, both should accept a task and produce a result.
	w1.Start(ctx)
	w2.Start(ctx)

	if err := w1.AddTask(task); err != nil {
		t.Fatalf("w1 AddTask failed: %v", err)
	}
	if err := w2.AddTask(task); err != nil {
		t.Fatalf("w2 AddTask failed: %v", err)
	}

	select {
	case r := <-w1.GetResults():
		if r != 42 {
			t.Fatalf("unexpected result from w1: %v", r)
		}
	case e := <-w1.GetErrors():
		t.Fatalf("unexpected error from w1: %v", e)
	}

	select {
	case r := <-w2.GetResults():
		if r != 42 {
			t.Fatalf("unexpected result from w2: %v", r)
		}
	case e := <-w2.GetErrors():
		t.Fatalf("unexpected error from w2: %v", e)
	}

	close(w1.GetResults())
	close(w1.GetErrors())
	close(w2.GetResults())
	close(w2.GetErrors())
}
