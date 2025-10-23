package workers

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ygrebnov/workers/pool"
)

func TestDispatcher_HappyPath(t *testing.T) {
	tasks := make(chan Task[int], 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	actualResults := make([]int, 0, 8)
	var inflight sync.WaitGroup
	rCh := make(chan int, 8)
	eCh := make(chan error, 8)
	evCh := make(chan completionEvent[int], 8)
	p := pool.NewDynamic(
		func() interface{} {
			return newWorker[int](rCh, eCh, false, false, evCh)
		})
	d := newDispatcher[int](tasks, &inflight, p)

	done := make(chan struct{})
	go func() { d.run(ctx); close(done) }()

	// enqueue a few tasks
	for i := 0; i < 5; i++ {
		v := i
		tasks <- TaskValue[int](func(context.Context) int { return v })
	}

	// give time to start all exec goroutines
	time.Sleep(50 * time.Millisecond)
	// cancel dispatcher and wait for it to exit
	cancel()
	<-done
	// wait for in-flight executions to complete
	inflight.Wait()

	// drain results
	close(rCh)
	close(eCh)
	close(evCh)
	for r := range rCh {
		actualResults = append(actualResults, r)
	}

	// verify all results executed (order not guaranteed)
	expectedResults := []int{0, 1, 2, 3, 4}

	if len(actualResults) != len(expectedResults) {
		t.Fatalf(
			"unexpected executed set length: got=%d want=%d",
			len(actualResults),
			len(expectedResults),
		)
	}

	sort.Ints(actualResults)
	for i := range expectedResults {
		if actualResults[i] != expectedResults[i] {
			t.Fatalf(
				"unexpected executed value at index %d: got=%d want=%d",
				i,
				actualResults[i],
				expectedResults[i],
			)
		}
	}
}

func TestDispatcher_CancelStopsReceiving(t *testing.T) {
	tasks := make(chan Task[int]) // unbuffered
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var countMu sync.Mutex
	execCount := 0
	execDone := make(chan struct{}, 1)
	rCh := make(chan int, 8)
	eCh := make(chan error, 8)
	evCh := make(chan completionEvent[int], 8)
	p := pool.NewDynamic(
		func() interface{} {
			return newWorker[int](rCh, eCh, false, false, evCh)
		})
	var inflight sync.WaitGroup
	d := newDispatcher[int](tasks, &inflight, p)

	done := make(chan struct{})
	go func() { d.run(ctx); close(done) }()

	// send first task and wait for exec
	tasks <- TaskValue[int](func(context.Context) int {
		countMu.Lock()
		execCount++
		countMu.Unlock()
		execDone <- struct{}{}
		return 1
	})
	select {
	case <-execDone:
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("executor did not process first task in time")
	}

	// cancel dispatcher
	cancel()
	<-done
	inflight.Wait()

	// try a non-blocking send; should not be received by dispatcher (already stopped)
	sent := false
	select {
	case tasks <- TaskValue[int](func(context.Context) int {
		countMu.Lock()
		execCount++
		countMu.Unlock()
		return 2
	}):
		sent = true
	default:
		// expected path: no receiver, send would block
	}
	if sent {
		t.Fatalf("task send unexpectedly succeeded after dispatcher was canceled")
	}

	// ensure only first task executed
	countMu.Lock()
	got := execCount
	countMu.Unlock()
	if got != 1 {
		t.Fatalf("unexpected exec count: got=%d want=1", got)
	}
}
