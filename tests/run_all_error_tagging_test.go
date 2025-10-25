package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

// unwrapJoined returns the slice of wrapped errors if err implements Unwrap() []error, otherwise returns err as a single-element slice.
func unwrapJoined(err error) []error {
	type unwrapper interface{ Unwrap() []error }
	if err == nil {
		return nil
	}
	if u, ok := err.(unwrapper); ok {
		return u.Unwrap()
	}
	return []error{err}
}

func TestRunAll_ErrorTagging_PreservesIDAndIndex(t *testing.T) {
	ctx := context.Background()

	tasks := []workers.Task[string]{
		workers.TaskErrorWithID[string]("a", func(context.Context) error { return errors.New("A") }),
		workers.TaskErrorWithID[string]("b", func(context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return errors.New("B")
		}),
		workers.TaskErrorWithID[string]("c", func(context.Context) error { return errors.New("C") }),
		workers.TaskErrorWithID[string]("d", func(context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return errors.New("D")
		}),
		workers.TaskErrorWithID[string]("e", func(context.Context) error { return errors.New("E") }),
	}

	res, err := workers.RunAll[string](
		ctx,
		tasks,
		workers.WithErrorTagging(),
		workers.WithStartImmediately(),
		workers.WithPreserveOrder(),
	)
	if len(res) != 0 {
		t.Fatalf("expected 0 results, got %d", len(res))
	}
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	parts := unwrapJoined(err)
	if len(parts) != 5 {
		t.Fatalf("expected 5 error parts, got %d", len(parts))
	}

	idsByIdx := map[int]any{}
	for _, e := range parts {
		id, ok := workers.ExtractTaskID(e)
		if !ok {
			t.Fatalf("missing id in error: %v", e)
		}
		idx, ok := workers.ExtractTaskIndex(e)
		if !ok {
			t.Fatalf("missing index in error: %v", e)
		}
		idsByIdx[idx] = id
	}

	if idsByIdx[0] != any("a") || idsByIdx[1] != any("b") || idsByIdx[2] != any("c") || idsByIdx[3] != any("d") || idsByIdx[4] != any("e") {
		t.Fatalf("unexpected ids by index: %+v", idsByIdx)
	}
}
