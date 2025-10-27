package tests

import (
	"runtime"
	"testing"

	"github.com/ygrebnov/workers"
)

func TestTasksBuffer(t *testing.T) {
	tests := []testCase{
		{
			name: "taskStringError_dynamic_startImmediately_buffer",
			options: []workers.Option{
				workers.WithStartImmediately(),
				workers.WithTasksBuffer(5), // zero value tested in nominal_test.go
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, false, false, false))
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []string{},
		},

		{
			name: "taskStringError_fixed_startImmediately_buffer",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
				workers.WithTasksBuffer(5), // zero value tested in nominal_test.go
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, false, false, false))
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []string{},
		},

		{
			name:   "taskString_dynamic_delayedStart_noBuffer",
			nTasks: 1,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, false, false, false))
			},
			expectedAddTaskError: &errAddTask{err: workers.ErrInvalidState.Error(), i: 1},
			expectedResults:      []string{},
			expectedErrors:       []string{},
			delayedStart:         true,
		},

		// No panic when tasks channel is full before Start.
		// We enqueue up to buffer capacity (4) before Start, then Start will drain.
		{
			name: "taskString_dynamic_delayedStart_buffered_OptionA",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithTasksBuffer(4),
			},
			nTasks: 4,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, false, false, false))
			},
			expectedResults: getExpectedResults(1, 2, 3, 4),
			expectedErrors:  []string{},
			delayedStart:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, testFn(&test))
	}
}
