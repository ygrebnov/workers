package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

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
		workers.TaskErrorWithID[string]("b", func(context.Context) error { return errors.New("B") }),
	}

	res, err := workers.RunAll[string](ctx, tasks, workers.WithErrorTagging(), workers.WithDynamicPool(), workers.WithStartImmediately())
	require.Len(t, res, 0)
	require.Error(t, err)

	parts := unwrapJoined(err)
	require.Len(t, parts, 2)

	idsByIdx := map[int]any{}
	for _, e := range parts {
		id, ok := workers.ExtractTaskID(e)
		require.True(t, ok)
		idx, ok := workers.ExtractTaskIndex(e)
		require.True(t, ok)
		idsByIdx[idx] = id
	}

	require.Equal(t, any("a"), idsByIdx[0])
	require.Equal(t, any("b"), idsByIdx[1])
}
