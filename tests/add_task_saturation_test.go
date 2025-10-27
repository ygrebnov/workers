package tests

import (
	"context"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

// Verifies that when TasksBufferSize > 0 and Start() has not been called yet,
// AddTask fills the buffer and a subsequent AddTask blocks (without panicking)
// until Start drains the queue. Ensures Option A (no panic on saturation).
func TestAddTask_BufferedBeforeStart_BlocksWithoutPanic_ThenStartUnblocks(t *testing.T) {
	ctx := context.Background()
	w, err := workers.New[int](
		ctx,
		workers.WithTasksBuffer(2), // small buffer to saturate easily
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Fill the buffer to capacity before Start.
	for i := 0; i < 2; i++ {
		if err := w.AddTask(workers.TaskValue[int](func(context.Context) int { return i })); err != nil {
			t.Fatalf("unexpected AddTask error while filling buffer: %v", err)
		}
	}

	// Start a goroutine that attempts to add one more task; this should block until Start.
	done := make(chan struct{})
	go func() {
		_ = w.AddTask(workers.TaskValue[int](func(context.Context) int { return 42 }))
		close(done)
	}()

	// Give it a brief moment to attempt the send; it should be blocked (not done yet).
	time.Sleep(50 * time.Millisecond)
	select {
	case <-done:
		t.Fatalf("AddTask should have been blocked before Start when buffer is full")
	default:
		// still blocked as expected
	}

	// Now start the workers; dispatcher starts draining, which should unblock the blocked AddTask.
	w.Start(ctx)

	select {
	case <-done:
		// unblocked as expected
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for blocked AddTask to unblock after Start")
	}
	// Wait for all 3 results before closing, to avoid racing Close() with the dispatcher
	// reading the third task from the queue.
	want := 3
	got := 0
	deadline := time.NewTimer(1 * time.Second)
	defer deadline.Stop()
	for got < want {
		select {
		case <-w.GetResults():
			got++
		case err := <-w.GetErrors():
			if err != nil {
				t.Fatalf("unexpected error received: %v", err)
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for %d results; got %d", want, got)
		}
	}

	// Close and ensure channels eventually close.
	w.Close()

	// Verify channels are closed.
	select {
	case _, ok := <-w.GetResults():
		if ok {
			t.Fatalf("results should be closed after Close()")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for results channel to close")
	}
	select {
	case _, ok := <-w.GetErrors():
		if ok {
			t.Fatalf("errors should be closed after Close()")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for errors channel to close")
	}
}
