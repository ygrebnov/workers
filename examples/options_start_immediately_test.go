package workers_test

import (
	"context"
	"fmt"

	"github.com/ygrebnov/workers"
)

// ExampleWithStartImmediately shows how to start the worker machinery right away when creating
// a Workers instance. Without this option, you must call Start(ctx) yourself before enqueuing tasks.
// - Use this when you want to enqueue tasks immediately after construction.
// - You can combine it with other options (fixed pool, buffers, etc.).
func ExampleWithStartImmediately() {
	ctx := context.Background()

	// New(..., WithStartImmediately()) constructs and starts the controller.
	w, _ := workers.New[int](ctx, workers.WithStartImmediately())

	// Now you can enqueue tasks right away; they will begin executing immediately.
	_ = w.AddTask(workers.TaskValue[int](func(context.Context) int { return 7 }))

	fmt.Println(<-w.GetResults()) // read the result

	// Release resources and close channels.
	w.Close()

	// Output: 7
}
