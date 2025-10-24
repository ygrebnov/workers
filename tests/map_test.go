package tests

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

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
	if err != nil {
		t.Fatalf("Map unordered failed: %v", err)
	}
	if len(resUnordered) != 3 {
		t.Fatalf("expected 3 results, got %d", len(resUnordered))
	}
	if resUnordered[0] == 0 {
		t.Fatalf("expected completion-order to not start with the slow item")
	}
	if resUnordered[2] != 0 {
		t.Fatalf("expected slow item to complete last in unordered mode")
	}

	// Preserve order: results should match input order exactly
	resOrdered, err := workers.Map[int, int](ctx, items, fn, workers.WithDynamicPool(), workers.WithStartImmediately(), workers.WithPreserveOrder())
	if err != nil {
		t.Fatalf("Map ordered failed: %v", err)
	}
	if !reflect.DeepEqual(items, resOrdered) {
		t.Fatalf("unexpected ordered results: got=%v want=%v", resOrdered, items)
	}
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
	if err == nil {
		t.Fatalf("expected error with StopOnError, got nil; results=%v", res)
	}
	// With StopOnError, after task 1 fails, the context is cancelled.
	// Some tasks may have already started and could complete or get cancelled.
	// The key validation is that we got an error (which we checked above).
	_ = res // Results are non-deterministic; error is what StopOnError guarantees
}
