package workers_test

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ygrebnov/workers"
)

// ExampleWithPreserveOrder shows how to keep results in the same order as inputs.
// By default, results are emitted as tasks finish (fastest first). If you need
// result i to always appear before result i+1, enable PreserveOrder.
// - This can reduce throughput a bit because results may wait for earlier ones.
// - Great for UI or reporting layers that expect predictable ordering.
func ExampleWithPreserveOrder() {
	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	w, _ := workers.New[string](ctx, workers.WithStartImmediately(), workers.WithPreserveOrder())

	// Add tasks that complete out of order; preserve-order will re-sequence outputs.
	for i := range 3 {
		_ = w.AddTask(workers.TaskValue[string](func(context.Context) string {
			// Later inputs finish sooner to demonstrate reordering
			time.Sleep(time.Duration(3-i) * 10 * time.Millisecond)
			return strconv.Itoa(i)
		}))
	}

	// Consume results.
	res := make([]string, 0, 3)
	for range 3 {
		select {
		case <-ctx.Done():
			// Context cancelled or timed out.
			fmt.Println(ctx.Err())
		case r := <-w.GetResults():
			res = append(res, r)
		case <-w.GetErrors():
			// Handle error.
		}
	}

	w.Close()

	// Print results in original input order.
	// Even though tasks finished out of order, PreserveOrder ensures correct sequencing.
	fmt.Println("results in order:", res)

	// Output: results in order: [0 1 2]
}
