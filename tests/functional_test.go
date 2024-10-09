package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func TestFunctional(t *testing.T) {
	tests := []struct {
		name            string
		maxWorkers      uint
		stopOnError     bool
		tasks           []interface{}
		expectedResults []string
		expectedErrors  []error
	}{
		{
			name:            "fixed, taskResultError, no errors",
			maxWorkers:      4,
			tasks:           []interface{}{basicTaskResultError, basicTaskResultError, basicTaskResultError},
			expectedResults: generateExpected(3, basicTaskResultError),
			expectedErrors:  nil,
		},
		{
			name:            "dynamic, taskResultError, no errors",
			maxWorkers:      0,
			tasks:           []interface{}{basicTaskResultError, basicTaskResultError, basicTaskResultError},
			expectedResults: generateExpected(3, basicTaskResultError),
			expectedErrors:  nil,
		},
		{
			name:            "fixed, taskResultError, many, no errors",
			maxWorkers:      4,
			tasks:           tasksN2048Size2,
			expectedResults: expectedN2048Size2,
			expectedErrors:  nil,
		},
		{
			name:            "dynamic, taskResultError, many, no errors",
			maxWorkers:      0,
			tasks:           tasksN2048Size2,
			expectedResults: expectedN2048Size2,
			expectedErrors:  nil,
		},
		{
			name:            "fixed, taskResultError error",
			maxWorkers:      4,
			tasks:           []interface{}{basicTaskResultError, errorTaskResultError, basicTaskResultError},
			expectedResults: generateExpected(2, basicTaskResultError),
			expectedErrors:  []error{errBasic},
		},
		{
			name:            "fixed, taskResultError error, stopOnError",
			maxWorkers:      4,
			stopOnError:     true,
			tasks:           []interface{}{longTaskResultError, errorTaskResultError, longTaskResultError, longTaskResultError},
			expectedResults: nil,
			expectedErrors:  []error{errBasic},
		},
		{
			name:            "dynamic, taskResultError error",
			maxWorkers:      0,
			tasks:           []interface{}{basicTaskResultError, errorTaskResultError, basicTaskResultError},
			expectedResults: generateExpected(2, basicTaskResultError),
			expectedErrors:  []error{errBasic},
		},
		{
			name:            "fixed, taskResultError panic",
			maxWorkers:      4,
			tasks:           []interface{}{basicTaskResultError, panicTaskResultError, basicTaskResultError},
			expectedResults: generateExpected(2, basicTaskResultError),
			expectedErrors:  []error{errPanic},
		},
		{
			name:            "dynamic, taskResultError panic",
			maxWorkers:      0,
			tasks:           []interface{}{basicTaskResultError, panicTaskResultError, basicTaskResultError},
			expectedResults: generateExpected(2, basicTaskResultError),
			expectedErrors:  []error{errPanic},
		},
		{
			name:            "dynamic, taskResultError panic, stopOnError",
			maxWorkers:      0,
			stopOnError:     true,
			tasks:           []interface{}{basicTaskResultError, panicTaskResultError, basicTaskResultError},
			expectedResults: nil,
			expectedErrors:  []error{errPanic},
		},

		// taskResult
		{
			name:            "fixed, taskResult, no errors",
			maxWorkers:      4,
			tasks:           []interface{}{basicTaskResult, basicTaskResult, basicTaskResult},
			expectedResults: generateExpected(3, basicTaskResult),
			expectedErrors:  nil,
		},
		{
			name:            "dynamic, taskResult, no errors",
			maxWorkers:      0,
			tasks:           []interface{}{basicTaskResult, basicTaskResult, basicTaskResult},
			expectedResults: generateExpected(3, basicTaskResult),
			expectedErrors:  nil,
		},
		{
			name:            "fixed, taskResult, panic",
			maxWorkers:      4,
			tasks:           []interface{}{basicTaskResult, panicTaskResult, basicTaskResult},
			expectedResults: generateExpected(2, basicTaskResult),
			expectedErrors:  []error{errPanic},
		},
		{
			name:            "dynamic, taskResult, panic",
			maxWorkers:      0,
			tasks:           []interface{}{basicTaskResult, panicTaskResult, basicTaskResult},
			expectedResults: generateExpected(2, basicTaskResult),
			expectedErrors:  []error{errPanic},
		},

		// taskError
		{
			name:            "fixed, taskError, no errors",
			maxWorkers:      4,
			tasks:           []interface{}{basicTaskError, basicTaskError, basicTaskError},
			expectedResults: nil,
			expectedErrors:  nil,
		},
		{
			name:            "dynamic, taskError, no errors",
			maxWorkers:      0,
			tasks:           []interface{}{basicTaskError, basicTaskError, basicTaskError},
			expectedResults: nil,
			expectedErrors:  nil,
		},
		{
			name:            "fixed, taskError, panic",
			maxWorkers:      4,
			tasks:           []interface{}{basicTaskError, panicLongTaskError, basicTaskError},
			expectedResults: nil,
			expectedErrors:  []error{errPanic},
		},
		{
			name:            "dynamic, taskError, panic",
			maxWorkers:      0,
			tasks:           []interface{}{basicTaskError, panicLongTaskError, basicTaskError},
			expectedResults: nil,
			expectedErrors:  []error{errPanic},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := workers.New[string](
				context.Background(),
				&workers.Config{
					MaxWorkers:       test.maxWorkers,
					StartImmediately: true,
					StopOnError:      test.stopOnError,
				},
			)

			done := make(chan struct{}, 1)

			go func() {
				actual := make([]string, 0, len(test.tasks))
				errors := make([]error, 0, len(test.tasks))

				for range len(test.expectedResults) + len(test.expectedErrors) {
					select {
					case result := <-w.GetResults():
						actual = append(actual, result)

					case err := <-w.GetErrors():
						errors = append(errors, err)
					}
				}

				require.ElementsMatch(t, actual, test.expectedResults)
				require.ElementsMatch(t, errors, test.expectedErrors)
				done <- struct{}{}
			}()

			for _, task := range test.tasks {
				err := w.AddTask(task)
				require.NoError(t, err)
			}

			<-done
		})
	}
}
