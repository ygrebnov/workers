package workers_test

import (
	"context"
	"runtime"
	"time"

	"github.com/ygrebnov/workers"
)

// ExampleWithIntakeChannel shows how to configure Workers to read tasks exclusively
// from a user-supplied channel. You own the channel's buffering and lifecycle.
// - Create a channel of Task[T] with your desired buffer size.
// - Pass it via WithIntakeChannel when constructing Workers.
// - Send tasks into the channel and close it when done.
func ExampleWithIntakeChannel() {
	ctx := context.Background()

	// User-provided intake channel; buffering is fully under your control.
	in := make(chan workers.Task[int], 2)

	// Construct Workers and start immediately. You can add more options as needed.
	w, _ := workers.New[int](
		ctx,
		workers.WithIntakeChannel(in),
		workers.WithFixedPool(uint(runtime.NumCPU())),
		workers.WithStartImmediately(),
	)

	// Send tasks to the intake channel and close when done.
	in <- workers.TaskValue[int](func(context.Context) int { time.Sleep(2 * time.Millisecond); return 1 })
	in <- workers.TaskValue[int](func(context.Context) int { return 2 })
	close(in)

	// In real code, range over w.GetResults()/w.GetErrors() until they are closed.
	_ = w.GetResults()
	_ = w.GetErrors()

	w.Close()

	// Output:
}
