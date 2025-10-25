package workers_test

import (
	"context"
	"errors"

	"github.com/ygrebnov/workers"
)

// ExampleWithErrorsBuffer shows how to increase the size of the outward errors channel.
// Why this matters: if your consumer reads errors slowly (or in bursts), a larger buffer
// prevents worker goroutines from blocking when they try to report errors.
// - WithErrorsBuffer(N) sets the size of the public GetErrors() channel when StopOnError is disabled.
// - If StopOnError is enabled, workers first write to a small internal buffer; see ExampleWithStopOnErrorBuffer.
func ExampleWithErrorsBuffer() {
	ctx := context.Background()

	// Increase error channel capacity so multiple errors can queue up without back-pressuring workers.
	w, _ := workers.New[int](ctx, workers.WithErrorsBuffer(4), workers.WithStartImmediately())

	// Add a task that fails. In your program, you should read from w.GetErrors().
	_ = w.AddTask(workers.TaskError[int](func(context.Context) error { return errors.New("boom") }))

	// For brevity, we don't read from the channel here, but in real code do:
	//   for e := range w.GetErrors() { ... }
	w.Close()
	// Output:
}
