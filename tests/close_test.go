package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func TestClose_Idempotent(t *testing.T) {
	ctx := context.Background()
	w := workers.New[int](ctx, &workers.Config{StartImmediately: true})

	// Call Close twice sequentially; should not panic.
	require.NotPanics(t, func() { w.Close() })
	require.NotPanics(t, func() { w.Close() })

	// Channels should be closed after Close returns.
	_, okR := <-w.GetResults()
	require.False(t, okR, "results channel should be closed after Close()")
	_, okE := <-w.GetErrors()
	require.False(t, okE, "errors channel should be closed after Close()")

	// Concurrent Close calls should also be safe and idempotent.
	w2 := workers.New[int](ctx, &workers.Config{StartImmediately: true})
	var wg sync.WaitGroup
	wg.Add(10)
	require.NotPanics(t, func() {
		for i := 0; i < 10; i++ {
			go func() { defer wg.Done(); w2.Close() }()
		}
		wg.Wait()
	})

	// Verify w2 channels are closed too.
	_, okR2 := <-w2.GetResults()
	require.False(t, okR2)
	_, okE2 := <-w2.GetErrors()
	require.False(t, okE2)
}

func TestAddTask_AfterClose_ReturnsInvalidState(t *testing.T) {
	ctx := context.Background()
	w := workers.New[int](ctx, &workers.Config{StartImmediately: true})

	w.Close()

	err := w.AddTask(workers.TaskValue[int](func(ctx context.Context) int { return 1 }))
	require.ErrorIs(t, err, workers.ErrInvalidState)
}
