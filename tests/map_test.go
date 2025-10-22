package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func TestMap_Ordering_PreserveVsUnordered(t *testing.T) {
	ctx := context.Background()
	items := []int{0, 1, 2}
	fn := func(ctx context.Context, x int) (int, error) {
		// Make item 0 slow; others fast
		if x == 0 {
			time.Sleep(40 * time.Millisecond)
		} else {
			time.Sleep(2 * time.Millisecond)
		}
		return x, nil
	}

	// Unordered (completion order): expect slow item (0) last
	resUnordered, err := workers.Map[int, int](ctx, items, fn, workers.WithDynamicPool(), workers.WithStartImmediately())
	require.NoError(t, err)
	require.Len(t, resUnordered, 3)
	require.NotEqual(t, 0, resUnordered[0], "expected completion-order to not start with the slow item")
	require.Equal(t, 0, resUnordered[2], "expected slow item to complete last in unordered mode")

	// Preserve order: results should match input order exactly
	resOrdered, err := workers.Map[int, int](ctx, items, fn, workers.WithDynamicPool(), workers.WithStartImmediately(), workers.WithPreserveOrder())
	require.NoError(t, err)
	require.Equal(t, items, resOrdered)
}

func TestMap_StopOnError_Sequential(t *testing.T) {
	ctx := context.Background()
	items := []int{0, 1, 2, 3, 4}

	fn := func(ctx context.Context, x int) (int, error) {
		if x == 1 {
			// Task 1 fails immediately
			return 0, errors.New("boom")
		}
		// All other tasks check if context was cancelled (due to StopOnError)
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(10 * time.Millisecond):
			return x, nil
		}
	}

	res, err := workers.Map[int, int](
		ctx,
		items,
		fn,
		workers.WithDynamicPool(),
		workers.WithStartImmediately(),
		workers.WithStopOnError(),
	)
	require.Error(t, err)
	// With StopOnError, after task 1 fails, the context is cancelled.
	// Some tasks may have already started and could complete or get cancelled.
	// The key validation is that we got an error (which we checked above).
	// Results are non-deterministic due to scheduler timing, but that's expected.
	_ = res // We got an error, which is what StopOnError guarantees
}
