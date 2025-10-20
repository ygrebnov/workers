package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func TestDefaultsParity_New_vs_NewOptions(t *testing.T) {
	ctx := context.Background()

	w1 := workers.New[int](ctx, nil)
	w2, err := workers.NewOptions[int](ctx)
	require.NoError(t, err)

	// By default, StartImmediately is false and TasksBufferSize is 0.
	// Adding a task before Start should return ErrInvalidState for both.
	task := func(context.Context) int { return 42 }
	require.ErrorIs(t, w1.AddTask(task), workers.ErrInvalidState)
	require.ErrorIs(t, w2.AddTask(task), workers.ErrInvalidState)

	// After Start, both should accept a task and produce a result.
	w1.Start(ctx)
	w2.Start(ctx)

	require.NoError(t, w1.AddTask(task))
	require.NoError(t, w2.AddTask(task))

	select {
	case r := <-w1.GetResults():
		require.Equal(t, 42, r)
	case e := <-w1.GetErrors():
		t.Fatalf("unexpected error from w1: %v", e)
	}

	select {
	case r := <-w2.GetResults():
		require.Equal(t, 42, r)
	case e := <-w2.GetErrors():
		t.Fatalf("unexpected error from w2: %v", e)
	}

	close(w1.GetResults())
	close(w1.GetErrors())
	close(w2.GetResults())
	close(w2.GetErrors())
}
