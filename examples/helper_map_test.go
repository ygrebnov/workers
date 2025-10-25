package workers_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ygrebnov/workers"
)

// ExampleMap shows how to transform a slice into another slice using the [workers.Map] function.
// In this example, we configure workers to preserve the input order in the output slice.
func ExampleMap() {
	ctx := context.Background()

	// Input slice to be transformed.
	in := []int{1, 2, 3, 4, 5}

	// The transformation function squares each number.
	// In this example, we simulate an error when processing the number 3.
	// WithPreserveOrder() ensures results are in the same order as inputs.
	res, err := workers.Map(
		ctx,
		in,
		func(ctx context.Context, x int) (string, error) {
			// Simulate an error for demonstration purposes.
			if x == 3 {
				return "", fmt.Errorf("simulated error on item %d", x)
			}
			return strconv.Itoa(x * x), nil
		},
		workers.WithPreserveOrder(),
	)
	if err != nil {
		// In real code, handle or log the error.
		fmt.Println("error:", err)
	}
	fmt.Println("results: [", strings.Join(res, ", "), "]")

	// Output: error: simulated error on item 3
	// results: [ 1, 4, 16, 25 ]
}
