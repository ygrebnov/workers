package tests

import (
	"context"
	"errors"
	"testing"
	"time"

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
		workers.WithStartImmediately(),
	)
	if err != nil {
		t.Fatalf("ForEach failed: %v", err)
	}
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
		workers.WithStartImmediately(),
		workers.WithStopOnError(),
	)
	if err == nil {
		t.Fatalf("expected error from ForEach with StopOnError, got nil")
	}
}

func TestForEach_Panic_PropagatesErrTaskPanicked(t *testing.T) {
	ctx := context.Background()
	items := []int{1}
	fn := func(ctx context.Context, _ int) error { panic("kaboom") }

	err := workers.ForEach[int](
		ctx,
		items,
		fn,
		workers.WithStartImmediately(),
	)
	if err == nil {
		t.Fatalf("expected error from panic, got nil")
	}
	if !errors.Is(err, workers.ErrTaskPanicked) {
		t.Fatalf("expected ErrTaskPanicked, got %v", err)
	}
}
