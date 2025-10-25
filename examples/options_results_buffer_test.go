package workers_test

import (
	"context"
	"fmt"

	"github.com/ygrebnov/workers"
)

// ExampleWithResultsBuffer shows how to increase the size of the results channel.
// Why this matters: if your consumer reads results slowly (e.g., doing extra work per item),
// a larger buffer helps avoid blocking worker goroutines that are trying to send results.
// - WithResultsBuffer(N) sets the capacity of the public GetResults() channel.
// - You should still read results promptly to avoid memory growth.
func ExampleWithResultsBuffer() {
	ctx := context.Background()

	// Increase results channel capacity so multiple results can queue up.
	w, _ := workers.New[int](ctx, workers.WithResultsBuffer(8), workers.WithStartImmediately())

	// Add a simple task that produces one result.
	_ = w.AddTask(workers.TaskValue[int](func(context.Context) int { return 10 }))

	fmt.Println(<-w.GetResults()) // read the result

	w.Close()

	// Output: 10
}
