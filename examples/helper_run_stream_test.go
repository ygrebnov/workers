package workers_test

import (
	"context"
	"runtime"
	"time"

	"github.com/ygrebnov/workers"
)

// ExampleRunStream shows how to execute a stream of Task values using a worker pool.
// - You create a channel of Task[T] and send tasks into it.
// - RunStream returns out (results) and errs (errors) channels.
// - Close the input channel when you're done sending tasks so processing can finish.
// - In real code, read from out/errs until they close.
func ExampleRunStream() {
	ctx := context.Background()

	// Input channel of tasks; buffered so the sender doesn't block immediately.
	in := make(chan workers.Task[int], 2)

	// Limit the workers pools size by the number of logical CPUs.
	// You can pass more options to tune behavior.
	out, errs, _ := workers.RunStream[int](ctx, in, workers.WithFixedPool(uint(runtime.NumCPU())))

	// Send a couple of tasks. TaskValue wraps a function that returns a value.
	in <- workers.TaskValue[int](func(context.Context) int { time.Sleep(2 * time.Millisecond); return 1 })
	in <- workers.TaskValue[int](func(context.Context) int { return 2 })
	close(in) // signal no more tasks

	// In order to range over results and errors, do:
	// for r := range out { ... } and for e := range errs { ... }.
	_ = out
	_ = errs

	// Output:
}
