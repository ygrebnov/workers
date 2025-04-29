package tests

import (
	"context"
	"runtime"
	"sort"
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
		results := make([]string, 0, 5)
		for range 5 {
			select {
			case <-timer.C:
				done <- struct{}{}
			case result := <-w.GetResults():
				results = append(results, result)
			case err := <-w.GetErrors():
				if err.Error() != "task execution panicked: panic on executing task for: 3" {
					t.Errorf("Unexpected message via errors channel: %s", err)
				}
			}
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got: %d", len(results))
		}

		sort.StringSlice(results).Sort()
		expectedResults := []string{
			"Executed for: 1, result: 1.",
			"Executed for: 2, result: 4.",
		}
		for i := range results {
			if results[i] != expectedResults[i] {
				t.Errorf("Expected result: %s, got: %s", expectedResults[i], results[i])
			}
		}

		done <- struct{}{}
		return
	}()

	for i := 1; i <= 5; i++ {
		err := w.AddTask(newTaskStringError(i, false, true))
		require.NoError(t, err)
	}

	<-done
}

func Test_Panic_DelayedStart_ContinueOnError_TaskString(t *testing.T) {
	w := workers.New[string](
		context.TODO(),
		&workers.Config{
			MaxWorkers:  uint(runtime.NumCPU()),
			StopOnError: false,
		},
	)

	done := make(chan struct{}, 1)

	go func() {
		timer := time.NewTimer(5 * time.Second)
		results := make([]string, 0, 5)
		for range 5 {
			select {
			case <-timer.C:
				t.Error("Timeout waiting for results")
				done <- struct{}{}
				return
			case result := <-w.GetResults():
				results = append(results, result)
			case err := <-w.GetErrors():
				if err.Error() != "task execution panicked: panic on executing task for: 3" {
					t.Errorf("Unexpected message via errors channel: %s", err)
				}
			}
		}

		if len(results) != 4 {
			t.Errorf("Expected 4 results, got: %d", len(results))
		}

		sort.StringSlice(results).Sort()
		expectedResults := []string{
			"Executed for: 1, result: 1.",
			"Executed for: 2, result: 4.",
			"Executed for: 4, result: 16.",
			"Executed for: 5, result: 25.",
		}
		for i := range results {
			if results[i] != expectedResults[i] {
				t.Errorf("Expected result: %s, got: %s", expectedResults[i], results[i])
			}
		}

		done <- struct{}{}
	}()

	w.Start(context.Background())

	for i := 1; i <= 5; i++ {
		err := w.AddTask(newTaskString(i, false, true))
		require.NoError(t, err)
	}

	<-done
}
