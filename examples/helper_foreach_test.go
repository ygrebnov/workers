package workers_test

import (
	"context"
	"fmt"

	"github.com/ygrebnov/workers"
)

// ExampleForEach shows how to run a function for each item in a slice using a pool of workers.
// In this example, we limit the workers pool size by 4 by passing WithFixedPool(4) option.
func ExampleForEach() {
	ctx := context.Background()

	// Items to apply the function to.
	items := []int{1, 2, 3, 4, 5}

	err := workers.ForEach(
		ctx,
		items,
		func(ctx context.Context, x int) error {
			// Simulate an error for demonstration purposes.
			if x == 3 {
				return fmt.Errorf("simulated error on item %d", x)
			}
			return nil
		},
		workers.WithFixedPool(4),
	)
	if err != nil {
		fmt.Println("error:", err)
	}

	// Output: error: simulated error on item 3
}
