package workers_test

import (
	"context"
	"fmt"

	"github.com/ygrebnov/workers"
)

// ExampleRunAll shows how to execute a batch of independent tasks and collect all results.
// RunAll owns the lifecycle for you: it creates a Workers instance, enqueues tasks,
// waits for them to finish, closes channels, and finally returns results and a joined error.
// - Results come in completion order (not the original order) unless you enable PreserveOrder.
// - If you enable StopOnError via options, the first error cancels the rest; some tasks may never run.
// - You can tune concurrency and buffers via options.
func ExampleRunAll() {
	ctx := context.Background()

	// Prepare a couple of tasks. TaskValue wraps a function that returns a value without error.
	tasks := []workers.Task[string]{
		workers.TaskValue(func(context.Context) string { return "a" }),
		workers.TaskValue(func(context.Context) string { return "b" }),
	}

	res, err := workers.RunAll[string](ctx, tasks)
	if err != nil {
		// If any task failed, err is errors.Join of all task errors.
		fmt.Println("error:", err)
	}
	_ = res // Use the results in your program; order is by completion.

	// Output:
}
