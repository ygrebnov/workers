package workers_test

import (
	"context"

	"github.com/ygrebnov/workers"
)

// ExampleMapStream shows how to transform a stream of inputs into outputs using a worker pool.
// This variant works with channels, so it's great for pipelines where data arrives over time.
func ExampleMapStream() {
	ctx := context.Background()

	// Create an input channel to send values into it.
	// Buffered input helps avoid blocking the sender while workers start up.
	in := make(chan int, 2)

	// MapStream returns two channels: out (results) and errs (errors).
	// The mapping function squares each number.
	// WithFixedPool(4) limits the workers pool size by 4.
	out, errs, _ := workers.MapStream(
		ctx,
		in,
		func(ctx context.Context, x int) (int, error) { return x * x, nil },
		workers.WithFixedPool(4),
	)

	// Send a couple of items, then close input to signal no more values will come.
	in <- 2
	in <- 3
	// Close the input channel when you're done sending, so the worker loop can finish.
	close(in)

	// Consume from out/errs to receive results and errors as they are produced.
	// In a real program you would range over out and errs until they're closed.
	// We ignore them here to keep the example short.
	_ = out
	_ = errs

	// Output:
}
