package workers

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// helper: receive from chan error with timeout
func recvErr(t *testing.T, ch <-chan error, d time.Duration) (error, bool) {
	t.Helper()
	select {
	case v := <-ch:
		return v, true
	case <-time.After(d):
		return nil, false
	}
}

// helper: check no value available on chan error non-blocking
func noRecvErr(t *testing.T, ch <-chan error) bool {
	t.Helper()
	select {
	case <-ch:
		return false
	default:
		return true
	}
}

// helper: check if a done channel is closed (non-blocking)
func isClosed(t *testing.T, ch <-chan struct{}) bool {
	t.Helper()
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestErrorForwarder_BufferedOut_ForwardsFirstAndCancelsFirst(t *testing.T) {
	in := make(chan error, 1)
	out := make(chan error, 1)
	closeCh := make(chan struct{})
	var sendWG sync.WaitGroup
	canceled := make(chan struct{})
	cancel := func() {
		select {
		case <-canceled:
		default:
			close(canceled)
		}
	}

	f := newErrorForwarder(in, out, closeCh, cancel, &sendWG)
	done := make(chan struct{})
	go func() { f.run(); close(done) }()

	in <- errors.New("boom")

	// Expect an error on out shortly
	v, ok := recvErr(t, out, 100*time.Millisecond)
	if !ok {
		t.Fatalf("expected forwarded error, got timeout")
	}
	if v == nil || v.Error() != "boom" {
		t.Fatalf("unexpected forwarded error: %v", v)
	}
	// By the time we observed it, cancel must have been called
	if !isClosed(t, canceled) {
		t.Fatalf("expected cancel to be called before/at forwarding")
	}
	close(closeCh)
	<-done
	sendWG.Wait()
}

func TestErrorForwarder_UnbufferedOut_UsesDetachedSenderAndDropsOnClose(t *testing.T) {
	in := make(chan error, 1)
	out := make(chan error) // unbuffered outward
	closeCh := make(chan struct{})
	var sendWG sync.WaitGroup
	canceled := make(chan struct{})
	cancel := func() {
		select {
		case <-canceled:
		default:
			close(canceled)
		}
	}

	f := newErrorForwarder(in, out, closeCh, cancel, &sendWG)
	done := make(chan struct{})
	go func() { f.run(); close(done) }()

	in <- errors.New("boom")

	// Give time for detached sender to be spawned (it will block on out send)
	time.Sleep(30 * time.Millisecond)
	// Do NOT attempt to read from out here; that would unblock the sender and deliver the value.
	// Instead, close to let detached sender drop and forwarder exit via closeCh path.
	close(closeCh)
	<-done
	sendWG.Wait()
	// No sends should be delivered after close
	if !noRecvErr(t, out) {
		t.Fatalf("unexpected error delivered after close")
	}
	// Cancel must have been called
	if !isClosed(t, canceled) {
		t.Fatalf("expected cancel to be called")
	}
}

func TestErrorForwarder_OnlyFirstForwarded_SubsequentDropped(t *testing.T) {
	in := make(chan error, 4)
	out := make(chan error, 4)
	closeCh := make(chan struct{})
	var sendWG sync.WaitGroup
	cancel := func() {}

	f := newErrorForwarder(in, out, closeCh, cancel, &sendWG)
	done := make(chan struct{})
	go func() { f.run(); close(done) }()

	in <- errors.New("first")
	in <- errors.New("second")
	in <- errors.New("third")

	// Expect only the first to appear
	v, ok := recvErr(t, out, 100*time.Millisecond)
	if !ok {
		t.Fatalf("expected first error to be forwarded")
	}
	if v == nil || v.Error() != "first" {
		t.Fatalf("unexpected first error: %v", v)
	}
	// Close to allow forwarder to drain and exit
	close(closeCh)
	<-done
	sendWG.Wait()
	// Ensure only one error forwarded
	if !noRecvErr(t, out) {
		t.Fatalf("expected only first error to be forwarded")
	}
}
