package tests

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/ygrebnov/workers"
)

func TestClose_Idempotent(t *testing.T) {
	ctx := context.Background()
	w := workers.New[int](ctx, &workers.Config{StartImmediately: true})

	// Call Close twice sequentially; should not panic.
	func() {
		deferred := false
		defer func() {
			if r := recover(); r != nil {
				deferred = true
				t.Fatalf("unexpected panic on Close(): %v", r)
			}
		}()
		_ = deferred
		w.Close()
	}()
	func() {
		deferred := false
		defer func() {
			if r := recover(); r != nil {
				deferred = true
				t.Fatalf("unexpected panic on Close(): %v", r)
			}
		}()
		_ = deferred
		w.Close()
	}()

	// Channels should be closed after Close returns.
	_, okR := <-w.GetResults()
	if okR {
		t.Fatalf("results channel should be closed after Close()")
	}
	_, okE := <-w.GetErrors()
	if okE {
		t.Fatalf("errors channel should be closed after Close()")
	}

	// Concurrent Close calls should also be safe and idempotent.
	w2 := workers.New[int](ctx, &workers.Config{StartImmediately: true})
	var wg sync.WaitGroup
	wg.Add(10)
	func() {
		deferred := false
		defer func() {
			if r := recover(); r != nil {
				deferred = true
				t.Fatalf("unexpected panic during concurrent Close(): %v", r)
			}
		}()
		_ = deferred
		for i := 0; i < 10; i++ {
			go func() { defer wg.Done(); w2.Close() }()
		}
		wg.Wait()
	}()

	// Verify w2 channels are closed too.
	_, okR2 := <-w2.GetResults()
	if okR2 {
		t.Fatalf("results channel should be closed after Close()")
	}
	_, okE2 := <-w2.GetErrors()
	if okE2 {
		t.Fatalf("errors channel should be closed after Close()")
	}
}

func TestAddTask_AfterClose_ReturnsInvalidState(t *testing.T) {
	ctx := context.Background()
	w := workers.New[int](ctx, &workers.Config{StartImmediately: true})

	w.Close()

	err := w.AddTask(workers.TaskValue[int](func(ctx context.Context) int { return 1 }))
	if !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
}
