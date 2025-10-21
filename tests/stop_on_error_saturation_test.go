package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

// TestStopOnError_CancelFirst_OutwardErrorsSaturated verifies that when StopOnError is enabled
// and the outward errors channel is not available (unbuffered and no reader), the system:
// 1) cancels promptly on the first error (further AddTask may fail immediately), and
// 2) does not deadlock even though forwarding the error would block (we forward asynchronously).
func TestStopOnError_CancelFirst_OutwardErrorsSaturated(t *testing.T) {
	t.Parallel()

	// Use a test-scoped timeout to avoid hangs.
	testCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Configure controller.
	w, err := workers.NewOptions[int](
		testCtx,
		workers.WithFixedPool(1),
		workers.WithStartImmediately(),
		workers.WithStopOnError(),
		workers.WithErrorsBuffer(0),      // outward errors channel unbuffered -> saturated without reader
		workers.WithStopOnErrorBuffer(1), // small internal buffer
		workers.WithTasksBuffer(4),       // small tasks buffer
	)
	if err != nil {
		t.Fatalf("failed to create workers via options: %v", err)
	}

	// Triggering task: returns error immediately.
	if err := w.AddTask(workers.TaskError[int](func(ctx context.Context) error { return errors.New("boom") })); err != nil {
		t.Fatalf("unexpected AddTask error for trigger task: %v", err)
	}

	// Wait briefly to allow cancel-first to propagate.
	time.Sleep(50 * time.Millisecond)

	// Attempt to enqueue a second task that would signal if started.
	started := make(chan struct{}, 1)
	err = w.AddTask(workers.TaskError[int](func(ctx context.Context) error {
		started <- struct{}{}
		return nil
	}))

	// Two acceptable outcomes:
	// - ErrInvalidState: cancellation already took effect and tasks channel is disabled.
	// - nil: task enqueued before tasks channel was disabled; it still must not start.
	if err != nil {
		if !errors.Is(err, workers.ErrInvalidState) {
			t.Fatalf("unexpected AddTask error after cancellation: %v", err)
		}
	} else {
		// Task was accepted; ensure it does not start within a short window.
		select {
		case <-started:
			t.Fatalf("second task started despite StopOnError cancellation")
		case <-time.After(200 * time.Millisecond):
			// success: did not start
		}
	}

	// Drain exactly one error to unblock the detached forwarder goroutine and avoid leaks.
	select {
	case <-w.GetErrors():
		// drained the triggering error
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting to drain outward error")
	}
}

// TestStopOnError_CancelFirst_BufferedOutward_NoDeadlock verifies that with a buffered outward
// errors channel (capacity > 0) and no reader, cancel-first still prevents deadlocks and
// further tasks do not start.
func TestStopOnError_CancelFirst_BufferedOutward_NoDeadlock(t *testing.T) {
	t.Parallel()

	testCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w, err := workers.NewOptions[int](
		testCtx,
		workers.WithFixedPool(1),
		workers.WithStartImmediately(),
		workers.WithStopOnError(),
		workers.WithErrorsBuffer(1),      // small buffered outward channel
		workers.WithStopOnErrorBuffer(1), // small internal buffer
		workers.WithTasksBuffer(4),
	)
	if err != nil {
		t.Fatalf("failed to create workers via options: %v", err)
	}

	if err := w.AddTask(workers.TaskError[int](func(ctx context.Context) error { return errors.New("boom") })); err != nil {
		t.Fatalf("unexpected AddTask error for trigger task: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	started := make(chan struct{}, 1)
	err = w.AddTask(workers.TaskError[int](func(ctx context.Context) error {
		started <- struct{}{}
		return nil
	}))

	if err != nil {
		if !errors.Is(err, workers.ErrInvalidState) {
			t.Fatalf("unexpected AddTask error after cancellation: %v", err)
		}
	} else {
		select {
		case <-started:
			t.Fatalf("second task started despite StopOnError cancellation")
		case <-time.After(200 * time.Millisecond):
			// success: did not start
		}
	}

	// Drain the single forwarded error to avoid leaking the detached forwarder goroutine (if any).
	select {
	case <-w.GetErrors():
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting to drain outward error")
	}
}

// TestOutwardErrors_BufferedOverflow_NoStopOnError stresses a buffered outward errors channel
// by enqueuing more failing tasks than the buffer capacity with StopOnError disabled. It verifies
// that even when the buffer is exceeded (senders block), draining proceeds and the system does not deadlock.
func TestOutwardErrors_BufferedOverflow_NoStopOnError(t *testing.T) {
	t.Parallel()

	testCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// StopOnError is NOT enabled here.
	const (
		poolSize   = 4
		bufSize    = 2
		tasksCount = 10
	)

	w, err := workers.NewOptions[int](
		testCtx,
		workers.WithFixedPool(poolSize),
		workers.WithStartImmediately(),
		workers.WithErrorsBuffer(bufSize),
		workers.WithTasksBuffer(tasksCount),
	)
	if err != nil {
		t.Fatalf("failed to create workers via options: %v", err)
	}

	// Enqueue more tasks than the outward errors buffer size; each returns an error immediately.
	for i := 0; i < tasksCount; i++ {
		if err := w.AddTask(workers.TaskError[int](func(ctx context.Context) error { return errors.New("boom") })); err != nil {
			t.Fatalf("unexpected AddTask error at %d: %v", i, err)
		}
	}

	// Give a brief moment for workers to run and fill the outward buffer; some workers may block on send.
	time.Sleep(20 * time.Millisecond)

	// Drain all errors; this should unblock any blocked workers and complete without deadlock.
	for i := 0; i < tasksCount; i++ {
		select {
		case <-w.GetErrors():
			// ok
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("timed out draining error %d/%d", i+1, tasksCount)
		}
	}
}
