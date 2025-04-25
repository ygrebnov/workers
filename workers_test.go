package workers

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type contextKey string

const key contextKey = "key"

func TestWorkers_Start_ThreadSafety(t *testing.T) {
	config := &Config{
		MaxWorkers:      5,
		TasksBufferSize: 10,
	}

	n := int32(10)
	ctxs := make([]context.Context, n)
	for i := range n {
		ctxs[i] = context.WithValue(context.Background(), key, fmt.Sprintf("value%d", i))
	}

	// initialize workers with context which is not in ctxs.
	w := New[string](context.Background(), config).(*workers[string])

	var wg sync.WaitGroup
	startCount := int32(0)

	// call Start multiple times concurrently.
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()

			w.Start(ctxs[i]) // each time different context.

			atomic.AddInt32(&startCount, 1)

			err := w.AddTask(func(ctx context.Context) (string, error) {
				val, ok := ctx.Value(key).(string)
				require.True(t, ok, "Expected value to be of type string")

				return val, nil
			})
			require.NoError(t, err, "Failed to add task to workers")
		}()
	}

	wg.Wait()

	if startCount != n {
		t.Errorf("expected Start to be called %d times, but got %d", n, startCount)
	}

	// check if all tasks were executed successfully and returned the same result.
	results := make(map[string]struct{})
	for range n {
		select {
		case r := <-w.GetResults():
			results[r] = struct{}{}

		case e := <-w.GetErrors():
			t.Errorf("unexpected error: %v", e)
		}
	}
	require.Len(t, results, 1, "Expected only one unique result, but got %d", len(results))

	close(w.GetResults())
	close(w.GetErrors())
}
