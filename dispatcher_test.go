package workers

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestDispatcher_HappyPath(t *testing.T) {
	tasks := make(chan Task[int], 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	seq := make([]int, 0, 8)
	exec := func(ctx context.Context, t Task[int]) {
		v, _ := t.Run(ctx)
		mu.Lock()
		seq = append(seq, v)
		mu.Unlock()
	}
	var inflight sync.WaitGroup
	d := newDispatcher[int](tasks, exec, &inflight)

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

	// verify all results executed (order not guaranteed)
	expected := []int{0, 1, 2, 3, 4}
	sort.Ints(seq)
	if !reflect.DeepEqual(seq, expected) {
		t.Fatalf("unexpected executed set: got=%v want=%v", seq, expected)
	}
}

func TestDispatcher_CancelStopsReceiving(t *testing.T) {
	tasks := make(chan Task[int]) // unbuffered
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var countMu sync.Mutex
	execCount := 0
	execDone := make(chan struct{}, 1)
	exec := func(ctx context.Context, t Task[int]) {
		_, _ = t.Run(ctx)
		countMu.Lock()
		execCount++
		countMu.Unlock()
		execDone <- struct{}{}
	}
	var inflight sync.WaitGroup
	d := newDispatcher[int](tasks, exec, &inflight)

	done := make(chan struct{})
	go func() { d.run(ctx); close(done) }()

	// send first task and wait for exec
	tasks <- TaskValue[int](func(context.Context) int { return 1 })
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
	case tasks <- TaskValue[int](func(context.Context) int { return 2 }):
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
