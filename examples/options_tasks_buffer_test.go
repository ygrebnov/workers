package workers_test

import (
	"context"

	"github.com/ygrebnov/workers"
)

// ExampleWithTasksBuffer shows how to allow enqueuing tasks before starting the worker machinery.
// By default, if you construct a Workers without starting it, AddTask would block on an unbuffered
// tasks channel. WithTasksBuffer(N) gives you a buffered tasks queue so you can enqueue up to N tasks
// before calling Start(ctx).
// Tips:
//   - This is handy when you want to prepare a batch, then start processing all at once.
//   - If the buffer fills (you added more than N), AddTask would block or panic depending on your usage;
//     keep buffer sizes reasonable for your workload.
//   - After Start(ctx), the dispatcher begins pulling tasks from the buffer and executing them.
//   - Close() waits for any started tasks to finish and then closes results/errors for you.
func ExampleWithTasksBuffer() {
	ctx := context.Background()

	// Create a controller with space for 2 tasks in the queue.
	w, _ := workers.New[int](ctx, workers.WithTasksBuffer(2))

	// Not started yet; tasks will queue in the buffer until Start is called.
	_ = w.AddTask(workers.TaskValue[int](func(context.Context) int { return 1 }))
	_ = w.AddTask(workers.TaskValue[int](func(context.Context) int { return 2 }))

	// Start workers — they will begin pulling from the queued tasks immediately.
	w.Start(ctx)

	// Consume results.
	for range 2 {
		select {
		case <-w.GetResults():
			// Handle result.
		case <-w.GetErrors():
			// Handle error.
		}
	}

	w.Close() // waits for in-flight tasks to complete and then closes channels

	// Output:
}
