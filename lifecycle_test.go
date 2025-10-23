package workers

import (
	"sync"
	"testing"
	"time"
)

// helper to read a string from a channel with timeout
func recvStep(t *testing.T, ch <-chan string, d time.Duration) (string, bool) {
	t.Helper()
	select {
	case s := <-ch:
		return s, true
	case <-time.After(d):
		return "", false
	}
}

func TestLifecycle_OrderAndSignals(t *testing.T) {
	steps := make(chan string, 10)

	// inflight starts at 1 so we control when shutdown proceeds beyond Wait
	var inflight sync.WaitGroup
	inflight.Add(1)

	closeCh := make(chan struct{})
	// observe closeCh closure
	closedObserved := make(chan struct{}, 1)
	go func() {
		<-closeCh
		steps <- "closeChClosed"
		closedObserved <- struct{}{}
	}()

	// stubs that record steps
	cancel := func() { steps <- "cancel" }
	drain := func() { steps <- "drainInternal" }
	closeEvents := func() { steps <- "closeEvents" }
	waitReorderer := func() { steps <- "waitReorderer" }
	closeResults := func() { steps <- "closeResults" }
	closeErrors := func() { steps <- "closeErrors" }

	lc := newLifecycleCoordinator(
		cancel,
		&inflight,
		closeCh,
		&sync.WaitGroup{}, // forwarderWG
		&sync.WaitGroup{}, // errorsSendWG
		drain,
		closeEvents,
		waitReorderer,
		closeResults,
		closeErrors,
	)

	done := make(chan struct{})
	go func() { lc.Close(); close(done) }()

	// First, we must see cancel before anything else
	if s, ok := recvStep(t, steps, 200*time.Millisecond); !ok || s != "cancel" {
		t.Fatalf("expected first step 'cancel', got=%q ok=%v", s, ok)
	}
	// closeCh must not be closed yet (blocked on inflight.Wait)
	select {
	case <-closedObserved:
		t.Fatalf("closeCh closed before inflight.Wait was released")
	default:
		// expected not yet closed
	}

	// Now allow shutdown to proceed
	inflight.Done()

	// Wait until we observe closeCh closure (do not assume it appears next in steps due to scheduling)
	select {
	case <-closedObserved:
		// observed
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected closeCh to be closed after inflight release")
	}

	// Now verify the remaining tail steps occur in order, skipping any interleaved 'closeChClosed' tokens
	expectedTail := []string{"drainInternal", "closeEvents", "waitReorderer", "closeResults", "closeErrors"}
	idx := 0
	deadline := time.After(500 * time.Millisecond)
	for idx < len(expectedTail) {
		select {
		case s := <-steps:
			if s == "closeChClosed" {
				continue
			}
			want := expectedTail[idx]
			if s != want {
				t.Fatalf("tail step %d: expected %q, got %q", idx+1, want, s)
			}
			idx++
		case <-deadline:
			t.Fatalf("timed out waiting for tail step %d (%q)", idx+1, expectedTail[idx])
		}
	}
	<-done
}

func TestLifecycle_Idempotent_ConcurrentClose(t *testing.T) {
	steps := make(chan string, 10)

	var inflight sync.WaitGroup
	closeCh := make(chan struct{})

	// observe closeCh closure exactly once
	closeChClosed := make(chan struct{}, 1)
	go func() {
		<-closeCh
		closeChClosed <- struct{}{}
	}()

	// stubs
	cancel := func() { steps <- "cancel" }
	drain := func() { steps <- "drainInternal" }
	closeEvents := func() { steps <- "closeEvents" }
	waitReorderer := func() { steps <- "waitReorderer" }
	closeResults := func() { steps <- "closeResults" }
	closeErrors := func() { steps <- "closeErrors" }

	lc := newLifecycleCoordinator(
		cancel,
		&inflight,
		closeCh,
		&sync.WaitGroup{},
		&sync.WaitGroup{},
		drain,
		closeEvents,
		waitReorderer,
		closeResults,
		closeErrors,
	)

	// call Close concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); lc.Close() }()
	}
	wg.Wait()

	// ensure closeCh closed once (observer receives exactly one signal)
	select {
	case <-closeChClosed:
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("closeCh was not closed")
	}
	// drain steps and ensure each expected token appears exactly once
	expected := map[string]int{
		"cancel":        0,
		"drainInternal": 0,
		"closeEvents":   0,
		"waitReorderer": 0,
		"closeResults":  0,
		"closeErrors":   0,
	}
	for {
		select {
		case s := <-steps:
			if _, ok := expected[s]; ok {
				expected[s]++
			}
		default:
			goto done
		}
	}

done:
	for k, v := range expected {
		if v != 1 {
			t.Fatalf("expected step %q exactly once, got %d", k, v)
		}
	}
}
