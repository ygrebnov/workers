package workers_test

import (
	"context"
	"runtime"
	"time"

	"github.com/ygrebnov/metrics"

	"github.com/ygrebnov/workers"
)

// ExampleWithMetrics shows how to configure Workers with a metrics provider.
// Here we use the built-in BasicProvider; in production you can pass your own adapter.
func ExampleWithMetrics() {
	ctx := context.Background()
	p := metrics.NewBasicProvider()

	w, _ := workers.New[int](
		ctx,
		workers.WithMetrics(p),
		workers.WithFixedPool(uint(runtime.NumCPU())),
		workers.WithStartImmediately(),
	)

	_ = w.AddTask(workers.TaskValue[int](func(context.Context) int { time.Sleep(2 * time.Millisecond); return 1 }))
	_ = w.AddTask(workers.TaskError[int](func(context.Context) error { return nil }))
	w.Close()

	// Inspect metrics if needed:
	_ = p.Counter("workers_tasks_completed_total")
	_ = p.Counter("workers_tasks_errors_total")
	_ = p.Histogram("workers_task_duration_seconds")

	// Output:
}
