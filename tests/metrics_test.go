package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
	"github.com/ygrebnov/workers/metrics"
)

func TestWithMetrics_BasicProvider_CountersAndHistogram(t *testing.T) {
	ctx := context.Background()
	p := metrics.NewBasicProvider()

	w, err := workers.New[int](ctx, workers.WithMetrics(p), workers.WithStartImmediately(), workers.WithFixedPool(2))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	nOK := 5
	nErr := 3
	total := nOK + nErr

	// enqueue successes
	for i := 0; i < nOK; i++ {
		_ = w.AddTask(workers.TaskValue[int](func(context.Context) int { time.Sleep(5 * time.Millisecond); return 1 }))
	}
	// enqueue errors
	for i := 0; i < nErr; i++ {
		_ = w.AddTask(workers.TaskError[int](func(context.Context) error { time.Sleep(3 * time.Millisecond); return errors.New("boom") }))
	}

	// drain results and errors
	gotOK := 0
	gotErr := 0
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for gotOK+gotErr < total {
		select {
		case _, ok := <-w.GetResults():
			if ok {
				gotOK++
			}
		case _, ok := <-w.GetErrors():
			if ok {
				gotErr++
			}
		case <-deadline.C:
			t.Fatalf("timeout waiting for results/errors; gotOK=%d gotErr=%d", gotOK, gotErr)
		}
	}
	w.Close()

	// verify metrics
	cCompleted := p.Counter("workers_tasks_completed_total").(*metrics.BasicCounter).Snapshot()
	cErrors := p.Counter("workers_tasks_errors_total").(*metrics.BasicCounter).Snapshot()
	h := p.Histogram("workers_task_duration_seconds").(*metrics.BasicHistogram).Snapshot()

	if int(cCompleted) != total {
		t.Fatalf("completed total = %d; want %d", cCompleted, total)
	}
	if int(cErrors) != nErr {
		t.Fatalf("errors total = %d; want %d", cErrors, nErr)
	}
	if h.Count != int64(total) {
		t.Fatalf("duration count = %d; want %d", h.Count, total)
	}
	if h.Min < 0 || h.Max < 0 {
		t.Fatalf("duration min/max must be non-negative; got (%v,%v)", h.Min, h.Max)
	}
}
