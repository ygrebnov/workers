package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func TestForEach_AllSucceed(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}
	fn := func(ctx context.Context, x int) error { return nil }

	err := workers.ForEach[int](
		ctx,
		items,
		fn,
		workers.WithDynamicPool(),
		workers.WithStartImmediately(),
	)
	require.NoError(t, err)
}

func TestForEach_StopOnError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	items := make([]int, 0, 32)
	for i := 0; i < 32; i++ {
		items = append(items, i)
	}
	fn := func(ctx context.Context, x int) error {
		if x == 10 {
			return errors.New("boom")
		}
		// small work to allow some scheduling
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Millisecond):
			return nil
		}
	}

	err := workers.ForEach[int](
		ctx,
		items,
		fn,
		workers.WithDynamicPool(),
		workers.WithStartImmediately(),
		workers.WithStopOnError(),
	)
	require.Error(t, err)
}

func TestForEach_Panic_PropagatesErrTaskPanicked(t *testing.T) {
	ctx := context.Background()
	items := []int{1}
	fn := func(ctx context.Context, _ int) error { panic("kaboom") }

	err := workers.ForEach[int](
		ctx,
		items,
		fn,
		workers.WithDynamicPool(),
		workers.WithStartImmediately(),
	)
	require.Error(t, err)
	require.True(t, errors.Is(err, workers.ErrTaskPanicked))
}
