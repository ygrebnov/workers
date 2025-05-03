package tests

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

type testCase struct {
	name                 string
	config               *workers.Config
	nTasks               int
	task                 func(int) interface{}
	expectedAddTaskError *errAddTask
	expectedResults      []string
	expectedErrors       []error
	contextTimeout       bool
	twoSetsOfTasks       bool
	delayedStart         bool
}

// getExpectedResults returns the expected results for the given task.
func getExpectedResults(results ...int) []string {
	expectedResults := make([]string, len(results))
	for i, result := range results {
		expectedResults[i] = fmt.Sprintf("Executed for: %d, result: %d.", result, result*result)
	}
	return expectedResults
}

type errAddTask struct {
	err        error
	shoudPanic bool
	i          int
}

func testFn(tc testCase) func(*testing.T) {
	return func(t *testing.T) {
		var cancel context.CancelFunc
		ctx := context.Background()

		if tc.contextTimeout {
			ctx, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel() // Ensure the context is canceled to release resources.
		}

		w := workers.New[string](ctx, tc.config)

		done := make(chan struct{}, 1)

		go func() {
			actualResults := make([]string, 0, tc.nTasks)
			actualErrors := make([]error, 0, tc.nTasks)
			timer := time.NewTimer(500 * time.Millisecond)

			for range tc.nTasks {
				select {
				case <-timer.C:
					checkResults(t, actualResults, tc.expectedResults, actualErrors, tc.expectedErrors)
					done <- struct{}{}
					return
				case result := <-w.GetResults():
					actualResults = append(actualResults, result)
				case err := <-w.GetErrors():
					actualErrors = append(actualErrors, err)
				}
			}

			checkResults(t, actualResults, tc.expectedResults, actualErrors, tc.expectedErrors)

			done <- struct{}{}
		}()

		for i := 1; i <= tc.nTasks; i++ {
			if tc.expectedAddTaskError != nil && tc.expectedAddTaskError.i == i {
				if tc.expectedAddTaskError.shoudPanic {
					require.Panics(t, func() { _ = w.AddTask(tc.task(i)) })
				} else {
					require.ErrorIs(t, w.AddTask(tc.task(i)), tc.expectedAddTaskError.err)
				}
			} else {
				err := w.AddTask(tc.task(i))
				require.NoError(t, err, "Failed to add task to workers")
			}
		}

		if tc.delayedStart {
			w.Start(ctx)
		}

		<-done

		if tc.twoSetsOfTasks {
			done2 := make(chan struct{}, 1)

			go func() {
				timer := time.NewTimer(200 * time.Millisecond)
				for range tc.nTasks {
					select {
					case <-timer.C:
						done2 <- struct{}{}
						return
					case result := <-w.GetResults():
						t.Errorf("Unexpected message via results channel: %v", result)
					case err := <-w.GetErrors():
						t.Errorf("Unexpected message via errors channel: %v", err)
					}
				}

				done2 <- struct{}{}
			}()

			for i := 1; i <= tc.nTasks; i++ {
				err := w.AddTask(tc.task(i))
				require.ErrorIs(t, err, workers.ErrInvalidState)
			}

			<-done2
			close(w.GetResults())
			close(w.GetErrors())
		} else {
			close(w.GetResults())
			close(w.GetErrors())
		}
	}
}

func checkResults(
	t *testing.T,
	actualResults []string,
	expectedResults []string,
	actualErrors []error,
	expectedErrors []error,
) {
	if len(actualResults) != len(expectedResults) {
		t.Errorf(
			"Expected %d results, got: %d\n%v (expected)\n%v (actual)",
			len(expectedResults),
			len(actualResults),
			expectedResults,
			actualResults,
		)
		return
	}

	sort.StringSlice(actualResults).Sort()
	for i := range actualResults {
		if actualResults[i] != expectedResults[i] {
			t.Errorf(
				"Elements with index %d do not match:\n%v (expected)\n%v (actual)",
				i,
				expectedResults,
				actualResults,
			)
			return
		}
	}

	if len(actualErrors) != len(expectedErrors) {
		t.Errorf(
			"Expected %d errors, got: %d\n%v (expected)\n%v (actual)",
			len(expectedErrors),
			len(actualErrors),
			expectedErrors,
			actualErrors,
		)
		return
	}

	sort.Slice(actualErrors, func(i, j int) bool {
		return actualErrors[i].Error() < actualErrors[j].Error()
	})
	for i := range actualErrors {
		if actualErrors[i].Error() != expectedErrors[i].Error() {
			t.Errorf(
				"Elements with index %d do not match:\n%v (expected)\n%v (actual)",
				i,
				expectedErrors,
				actualErrors,
			)
			return
		}
	}
}
