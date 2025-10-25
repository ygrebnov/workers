package workers_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ygrebnov/workers"
)

// ExampleWithStopOnError shows how to cancel the whole batch on the first error.
// When StopOnError is enabled:
// - The first error triggers cancellation; tasks that haven't started yet won't run.
// - Workers report errors through the outward errors channel; you should consume it.
// - Consider also tuning buffers with WithStopOnErrorBuffer and WithErrorsBuffer.
func ExampleWithStopOnError() {
	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Enable StopOnError so the first failing task cancels the rest.
	w, _ := workers.New[string](
		ctx,
		workers.WithStartImmediately(),
		workers.WithStopOnError(),
		workers.WithPreserveOrder(),
	)

	// Prepare a task generator that fails on task 3.
	task := func(i int) func(context.Context) (string, error) {
		return func(context.Context) (string, error) {
			time.Sleep(time.Duration(20*i) * time.Millisecond) // Simulate work

			if i == 3 {
				return "", errors.New("task execution error")
			}

			return fmt.Sprintf("task %d completed", i), nil
		}
	}

	// Add tasks 1 to 5. Task 3 will fail and cancel the rest.
	for i := range 5 {
		_ = w.AddTask(workers.TaskFunc[string](task(i + 1)))
	}

	res := make([]string, 0, 5)
	errs := make([]string, 0, 5)

	getResults := func() {
		for {
			select {
			case <-ctx.Done():
				// Context cancelled or timed out.
				fmt.Println(ctx.Err())
				return
			case r := <-w.GetResults():
				res = append(res, r)
			case e := <-w.GetErrors():
				// Handle error.
				errs = append(errs, fmt.Sprintf("error received: %v", e))
			}
		}
	}

	getResults()
	fmt.Println(strings.Join(res, ", "))
	fmt.Println(strings.Join(errs, ", "))

	w.Close()

	// Output: context deadline exceeded
	// task 1 completed, task 2 completed
	// error received: task execution error

}
