package tests

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

type testWorkers[R any] interface {
	Start(context.Context)
	AddTask(workers.Task[R]) error
	GetResults() chan R
	GetErrors() chan error
}

type testCase struct {
	name                 string
	options              []workers.Option
	nTasks               int
	task                 func(int) workers.Task[string]
	expectedAddTaskError *errAddTask
	expectedMaxResults   *int
	expectedResults      []string
	expectedErrors       []string
	contextTimeout       bool
	twoSetsOfTasks       bool
	delayedStart         bool
}

func ptrInt(i int) *int {
	return &i
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
	err         string
	shouldPanic bool
	i           int
}

func testFn(tc *testCase) func(*testing.T) {
	return func(t *testing.T) {
		var cancel context.CancelFunc
		ctx := context.Background()

		if tc.contextTimeout {
			ctx, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel() // Ensure the context is canceled to release resources.
		}

		w, err := workers.New[string](ctx, tc.options...)
		if err != nil {
			t.Fatalf("New failed: %v", err)
		}

		done := make(chan struct{}, 1)

		go func() {
			actualResults := make([]string, 0, tc.nTasks)
			actualErrors := make([]error, 0, tc.nTasks)
			timer := time.NewTimer(500 * time.Millisecond)

			for range tc.nTasks {
				select {
				case <-timer.C:
					checkResults(t, actualResults, tc.expectedMaxResults, tc.expectedResults, actualErrors, tc.expectedErrors)
					done <- struct{}{}
					return
				case result := <-w.GetResults():
					actualResults = append(actualResults, result)
				case err := <-w.GetErrors():
					actualErrors = append(actualErrors, err)
				}
			}

			checkResults(t, actualResults, tc.expectedMaxResults, tc.expectedResults, actualErrors, tc.expectedErrors)

			done <- struct{}{}
		}()

		for i := 1; i <= tc.nTasks; i++ {
			if tc.expectedAddTaskError != nil && tc.expectedAddTaskError.i == i {
				if tc.expectedAddTaskError.shouldPanic {
					func() {
						deferred := false
						defer func() {
							if r := recover(); r == nil {
								deferred = true
								t.Errorf("expected panic when adding task %d, got none", i)
							}
						}()
						_ = w.AddTask(tc.task(i))
						_ = deferred
					}()
				} else {
					err := w.AddTask(tc.task(i))
					if err == nil || !strings.Contains(err.Error(), tc.expectedAddTaskError.err) {
						t.Fatalf("expected error containing %q when adding task %d, got %v", tc.expectedAddTaskError.err, i, err)
					}
				}
			} else {
				err := w.AddTask(tc.task(i))
				if err != nil {
					t.Fatalf("Failed to add task to workers: %v", err)
				}
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
				if !errors.Is(err, workers.ErrInvalidState) {
					t.Fatalf("expected ErrInvalidState when adding task %d, got %v", i, err)
				}
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
	expectedMaxResults *int,
	expectedResults []string,
	actualErrors []error,
	expectedErrors []string,
) {
	switch {
	case expectedMaxResults != nil && len(actualResults) > *expectedMaxResults:
		t.Errorf(
			"Expected max %d results, got: %d\n%v (expected)\n%v (actual)",
			*expectedMaxResults,
			len(actualResults),
			expectedResults,
			actualResults,
		)
		return
	case expectedMaxResults == nil:
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
		if actualErrors[i].Error() != expectedErrors[i] {
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

func executionErrorMsg(i int) string {
	return fmt.Sprintf("error executing task for: %d", i)
}

func panicErrorMsg(i int) string {
	return fmt.Sprintf("workers: task execution panicked: panic on executing task for: %d", i)
}
