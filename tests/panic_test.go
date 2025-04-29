package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func Test_Panic_TaskStringError(t *testing.T) {
	w := workers.New[string](
		context.Background(),
		&workers.Config{
			StartImmediately: true,
			StopOnError:      true,
		},
	)

	done := make(chan struct{}, 1)

	go func() {
		timer := time.NewTimer(5 * time.Second)
		results := make([]string, 0, 10)
		for range 10 {
			select {
			case <-timer.C:
				done <- struct{}{}
			case result := <-w.GetResults():
				results = append(results, result)
			case err := <-w.GetErrors():
				if err.Error() != "error executing task for: 3" {
					t.Errorf("Unexpected message via errors channel: %s", err)
				}
			}
		}

		require.Len(t, results, 2, "Expected 2 results")
		require.ElementsMatch(t, results, []string{
			"Executed for: 1, result: 1.",
			"Executed for: 2, result: 4.",
		})

		done <- struct{}{}
		return
	}()

	for i := 1; i <= 10; i++ {
		err := w.AddTask(newTaskStringError(i, false, true))
		require.NoError(t, err)
	}

	<-done
}
