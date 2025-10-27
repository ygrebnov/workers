package tests

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

func TestIntakeChannel_AddApisRejected(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int], 1)
	w, err := workers.New[int](ctx, workers.WithIntakeChannel(in))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// AddTask API family must be rejected in intake-channel mode.
	if err := w.AddTask(workers.TaskValue[int](func(context.Context) int { return 1 })); !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("AddTask should be ErrInvalidState, got %v", err)
	}
	ok, err := w.TryAddTask(workers.TaskValue[int](func(context.Context) int { return 1 }))
	if ok || !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("TryAddTask should be (false, ErrInvalidState), got (%v, %v)", ok, err)
	}
	callCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if err := w.AddTaskContext(callCtx, workers.TaskValue[int](func(context.Context) int { return 1 })); !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("AddTaskContext should be ErrInvalidState, got %v", err)
	}
}

func TestIntakeChannel_PreStart_Buffered_DrainsOnStart(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int], 2)
	w, err := workers.New[int](ctx, workers.WithIntakeChannel(in))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Preload two tasks into intake buffer before Start.
	in <- workers.TaskValue[int](func(context.Context) int { return 10 })
	in <- workers.TaskValue[int](func(context.Context) int { return 20 })

	w.Start(ctx)

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

func TestIntakeChannel_AfterStart_Forwarding(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int])
	w, err := workers.New[int](ctx, workers.WithIntakeChannel(in), workers.WithStartImmediately())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	go func() {
		in <- workers.TaskValue[int](func(context.Context) int { return 1 })
		in <- workers.TaskValue[int](func(context.Context) int { return 2 })
		in <- workers.TaskValue[int](func(context.Context) int { return 3 })
		close(in)
	}()

	recv := make([]int, 0, 3)
	deadline := time.NewTimer(1 * time.Second)
	defer deadline.Stop()
	for len(recv) < 3 {
		select {
		case v := <-w.GetResults():
			recv = append(recv, v)
		case err := <-w.GetErrors():
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for results; got %v", recv)
		}
	}
	w.Close()

	sort.Ints(recv)
	if !reflect.DeepEqual(recv, []int{1, 2, 3}) {
		t.Fatalf("unexpected results: %v", recv)
	}
}

func TestIntakeChannel_PreserveOrder(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int])
	w, err := workers.New[int](
		ctx,
		workers.WithIntakeChannel(in),
		workers.WithPreserveOrder(),
		workers.WithStartImmediately(),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	go func() {
		// Send three tasks that complete out of order.
		in <- workers.TaskValue[int](func(context.Context) int { time.Sleep(80 * time.Millisecond); return 1 }) // slow, index 0
		in <- workers.TaskValue[int](func(context.Context) int { time.Sleep(10 * time.Millisecond); return 2 }) // fast, index 1
		in <- workers.TaskValue[int](func(context.Context) int { time.Sleep(20 * time.Millisecond); return 3 }) // medium, index 2
		close(in)
	}()

	recv := make([]int, 0, 3)
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for len(recv) < 3 {
		select {
		case v := <-w.GetResults():
			recv = append(recv, v)
		case err := <-w.GetErrors():
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for ordered results; got %v", recv)
		}
	}

	w.Close()

	// Must be [1,2,3] due to preserve-order.
	if !reflect.DeepEqual(recv, []int{1, 2, 3}) {
		t.Fatalf("results are not in input order: %v", recv)
	}
}

func TestIntakeChannel_StopOnError_Cancels(t *testing.T) {
	ctx := context.Background()
	in := make(chan workers.Task[int], 1)
	w, err := workers.New[int](
		ctx,
		workers.WithIntakeChannel(in),
		workers.WithStopOnError(),
		workers.WithStopOnErrorBuffer(1),
		workers.WithErrorsBuffer(1),
		workers.WithStartImmediately(),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Trigger first error; this should cancel the controller.
	in <- workers.TaskError[int](func(context.Context) error { return errors.New("boom") })

	select {
	case <-w.GetErrors():
		// ok, cancellation propagated
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for outward error")
	}

	// Post-cancel, AddTask APIs should be ErrExclusiveIntakeChannel.
	if err := w.AddTask(workers.TaskValue[int](func(context.Context) int { return 7 })); !errors.Is(err, workers.ErrExclusiveIntakeChannel) {
		t.Fatalf("expected AddTask ErrExclusiveIntakeChannel after cancel, got %v", err)
	}
	ok, err := w.TryAddTask(workers.TaskValue[int](func(context.Context) int { return 8 }))
	if ok || !errors.Is(err, workers.ErrExclusiveIntakeChannel) {
		t.Fatalf("expected TryAddTask (false, ErrExclusiveIntakeChannel) after cancel, got (%v, %v)", ok, err)
	}
	callCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if err := w.AddTaskContext(callCtx, workers.TaskValue[int](func(context.Context) int { return 9 })); !errors.Is(err, workers.ErrExclusiveIntakeChannel) {
		t.Fatalf("expected AddTaskContext ErrExclusiveIntakeChannel after cancel, got %v", err)
	}

	w.Close()
}

func TestIntakeChannel_TypeMismatch_ReturnsInvalidConfig(t *testing.T) {
	ctx := context.Background()
	// Provide channel for a mismatched R (string) while constructing Workers[int].
	in := make(chan workers.Task[string])
	w, err := workers.New[int](ctx, workers.WithIntakeChannel(in))
	if err == nil || !errors.Is(err, workers.ErrInvalidConfig) || w != nil {
		t.Fatalf("expected ErrInvalidConfig and nil workers on type mismatch; got (w=%v, err=%v)", w, err)
	}
}
