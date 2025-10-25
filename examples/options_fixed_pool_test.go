package workers_test

import (
	"context"
	"fmt"
	"time"

	"github.com/ygrebnov/workers"
)

// ExampleWithFixedPool shows how to cap parallelism with a fixed-size worker pool.
// Why use it: limit concurrency to avoid overloading downstream services/CPU.
// - WithFixedPool(N) creates at most N workers; extra tasks wait until a worker is free.
// - You can still buffer tasks/results/errors independently.
// - It's good practice to consume results before Close to let senders finish cleanly.
func ExampleWithFixedPool() {
	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	w, _ := workers.New[int](ctx, workers.WithFixedPool(4), workers.WithStartImmediately())

	// Enqueue two simple tasks. The second sleeps a bit to simulate work.
	_ = w.AddTask(workers.TaskValue[int](
		func(context.Context) int {
			return 1
		},
	))
	_ = w.AddTask(workers.TaskValue[int](
		func(context.Context) int {
			time.Sleep(10 * time.Millisecond)
			return 2
		},
	))

	// Consume results.
	for range 2 {
		select {
		case <-ctx.Done():
			// Context cancelled or timed out.
			fmt.Println(ctx.Err())
		case <-w.GetResults():
			// Handle result.
		case <-w.GetErrors():
			// Handle error.
		}
	}

	w.Close()

	// Output:
}
