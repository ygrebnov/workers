package workers_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/ygrebnov/workers"
)

// ExampleWithErrorTagging shows how to enable error tagging so every error carries
// extra metadata: the task ID (if you set one) and the input index (its position in the batch).
// This helps correlate a failure back to the exact input that caused it.
// - Enable with workers.WithErrorTagging().
// - When you read from GetErrors(), use ExtractTaskID and ExtractTaskIndex to retrieve metadata.
func ExampleWithErrorTagging() {
	ctx := context.Background()

	// Start immediately and turn on error tagging. You can combine this with other options.
	w, _ := workers.New[string](
		ctx,
		workers.WithStartImmediately(),
		workers.WithErrorTagging(),
	)

	// Add a task that fails. We also give it a stable ID so we can recognize it in logs/metrics.
	_ = w.AddTask(
		workers.TaskErrorWithID[string](
			"id-1",
			func(context.Context) error { return errors.New("task execution error") },
		),
	)

	// Read and inspect metadata (in real code, you would range until the channel closes).
	select {
	case e := <-w.GetErrors():
		if id, ok := workers.ExtractTaskID(e); ok {
			fmt.Printf("Task execution error, task_id: %v, error: %v\n", id, e)
		}
		if idx, ok := workers.ExtractTaskIndex(e); ok {
			fmt.Printf("Task execution error, task_index: %d, error: %v", idx, e)
		}
	}

	w.Close()

	// Output: Task execution error, task_id: id-1, error: task execution error
	// Task execution error, task_index: 0, error: task execution error
}
