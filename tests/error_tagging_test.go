package tests

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

func TestErrorTagging_Disabled_NoMeta(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[string](ctx, workers.WithStartImmediately())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	n := 5
	msgs := make([]string, n)
	for i := 0; i < n; i++ {
		msg := fmt.Sprintf("e%d", i)
		msgs[i] = msg
		if err := w.AddTask(workers.TaskError[string](func(context.Context) error { return errors.New(msg) })); err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
	}

	received := make([]string, 0, n)
	for i := 0; i < n; i++ {
		select {
		case e := <-w.GetErrors():
			received = append(received, e.Error())
			id, ok := workers.ExtractTaskID(e)
			if ok {
				t.Fatalf("unexpected task id present: %v", id)
			}
			_, ok = workers.ExtractTaskIndex(e)
			if ok {
				t.Fatalf("unexpected task index present")
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for error %d", i)
		}
	}

	sort.Strings(received)
	sort.Strings(msgs)
	if len(received) != len(msgs) {
		t.Fatalf("unexpected errors count: got=%d want=%d", len(received), len(msgs))
	}
	for i := range msgs {
		if msgs[i] != received[i] {
			t.Fatalf("unexpected error at %d: got=%q want=%q", i, received[i], msgs[i])
		}
	}

	w.Close()
}

func TestErrorTagging_Enabled_AssignsIDAndIndex(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[string](ctx, workers.WithStartImmediately(), workers.WithErrorTagging())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	n := 5
	ids := make([]any, n)
	for i := 0; i < n; i++ {
		ids[i] = fmt.Sprintf("id-%d", i)
		if err := w.AddTask(workers.TaskErrorWithID[string](ids[i], func(context.Context) error { return fmt.Errorf("boom-%d", i) })); err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
	}

	recv := 0
	seen := map[int]any{}
	for recv < n {
		select {
		case e := <-w.GetErrors():
			id, ok := workers.ExtractTaskID(e)
			if !ok {
				t.Fatalf("missing task id in error: %v", e)
			}
			idx, ok := workers.ExtractTaskIndex(e)
			if !ok {
				t.Fatalf("missing task index in error: %v", e)
			}
			seen[idx] = id
			recv++
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for error %d", recv)
		}
	}

	for i := 0; i < n; i++ {
		if ids[i] != seen[i] {
			t.Fatalf("id mismatch for index %d: got=%v want=%v", i, seen[i], ids[i])
		}
	}

	w.Close()
}

func TestErrorTagging_StopOnError_FirstTagged(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[string](
		ctx,
		workers.WithStartImmediately(),
		workers.WithErrorTagging(),
		workers.WithStopOnError(),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	firstID := "first"
	secondID := "second"

	if err := w.AddTask(workers.TaskErrorWithID[string](firstID, func(context.Context) error { return fmt.Errorf("first-err") })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	// Second AddTask may fail if cancellation is already triggered; tolerate ErrInvalidState.
	_ = w.AddTask(workers.TaskErrorWithID[string](secondID, func(context.Context) error { return fmt.Errorf("second-err") }))

	select {
	case e := <-w.GetErrors():
		id, ok := workers.ExtractTaskID(e)
		if !ok {
			t.Fatalf("missing id in error: %v", e)
		}
		if !(id == any(firstID) || id == any(secondID)) {
			t.Fatalf("unexpected id: %v", id)
		}
		idx, ok := workers.ExtractTaskIndex(e)
		if !ok {
			t.Fatalf("missing index in error: %v", e)
		}
		if idx != 0 && idx != 1 {
			t.Fatalf("unexpected index: %d", idx)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first error")
	}

	w.Close()
}

func TestErrorTagging_Panic_Tagged(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](ctx, workers.WithStartImmediately(), workers.WithErrorTagging())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	panicID := "panic-task"
	if err := w.AddTask(workers.TaskValueWithID[int](panicID, func(context.Context) int { panic("boom") })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	select {
	case e := <-w.GetErrors():
		if !errors.Is(e, workers.ErrTaskPanicked) {
			t.Fatalf("expected ErrTaskPanicked, got %v", e)
		}
		id, ok := workers.ExtractTaskID(e)
		if !ok {
			t.Fatalf("missing id on panic error: %v", e)
		}
		if id != any(panicID) {
			t.Fatalf("unexpected id: got=%v want=%v", id, panicID)
		}
		idx, ok := workers.ExtractTaskIndex(e)
		if !ok {
			t.Fatalf("missing index on panic error: %v", e)
		}
		if idx != 0 {
			t.Fatalf("unexpected index: got=%d want=0", idx)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for panic error")
	}

	w.Close()
}

func TestErrorTagging_Cancel_Tagged(t *testing.T) {
	outerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w, err := workers.New[string](outerCtx, workers.WithStartImmediately(), workers.WithErrorTagging())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	cancelID := "cancel-task"
	started := make(chan struct{}, 1)
	// Task waits for ctx.Done then returns a concrete error, and signals when started.
	blocker := workers.TaskErrorWithID[string](cancelID, func(ctx context.Context) error {
		select {
		case started <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return fmt.Errorf("canceled: %v", ctx.Err())
	})

	if err := w.AddTask(blocker); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Ensure the task started before cancel.
	select {
	case <-started:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("task did not start in time")
	}

	// Cancel the outer context to trigger internal cancellation without closing channels yet.
	cancel()

	select {
	case e := <-w.GetErrors():
		id, ok := workers.ExtractTaskID(e)
		if !ok {
			t.Fatalf("received untagged error: %T: %+v", e, e)
		}
		if id != any(cancelID) {
			t.Fatalf("unexpected id: got=%v want=%v", id, cancelID)
		}
		idx, ok := workers.ExtractTaskIndex(e)
		if !ok {
			t.Fatalf("received error without index: %T: %+v", e, e)
		}
		if idx != 0 {
			t.Fatalf("unexpected index: got=%d want=0", idx)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cancel error")
	}

	w.Close()
}
