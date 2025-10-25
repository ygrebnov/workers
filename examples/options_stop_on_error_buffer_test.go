package workers_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/ygrebnov/workers"
)

// ExampleWithStopOnErrorBuffer shows how to size the internal error buffer used in StopOnError mode.
// When StopOnError is enabled, workers first write errors to a small internal channel (errorsBuf).
// The library forwards only the first error to the outward errors channel and then cancels the context.
// Sizing details:
//   - WithStopOnErrorBuffer(N) controls the internal buffer workers write to before cancellation propagates.
//     A slightly larger buffer can reduce the chance that a worker blocks when many errors happen fast.
//   - WithErrorsBuffer(M) controls the public outward errors channel capacity (useful for your consumer speed).
func ExampleWithStopOnErrorBuffer() {
	ctx := context.Background()

	// Start immediately, enable StopOnError, and tune both the internal and outward error buffers.
	w, _ := workers.New[int](ctx,
		workers.WithStartImmediately(),
		workers.WithStopOnError(),
		workers.WithStopOnErrorBuffer(1), // internal buffer for worker->forwarder
		workers.WithErrorsBuffer(1),      // outward buffer for your reader
	)

	// Add a failing task to trigger cancellation. In your code, read errors/results until close.
	_ = w.AddTask(workers.TaskError[int](func(context.Context) error { return errors.New("error") }))

	fmt.Println(<-w.GetErrors()) // read the error

	w.Close()

	// Output: error
}
