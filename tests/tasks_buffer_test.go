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
			config: &workers.Config{
				StartImmediately: true,
				TasksBufferSize:  5, // zero value tested in nominal_test.go
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, false, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []error{},
		},

		{
			name: "taskStringError_fixed_startImmediately_buffer",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
				TasksBufferSize:  5, // zero value tested in nominal_test.go
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, false, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []error{},
		},

		{
			name: "taskString_dynamic_delayedStart_noBuffer",
			config: &workers.Config{
				MaxWorkers: uint(runtime.NumCPU()),
			},
			nTasks: 1,
			task: func(i int) interface{} {
				return newTaskString(i, false, false, false)
			},
			expectedAddTaskError: &errAddTask{err: workers.ErrInvalidState, i: 1},
			expectedResults:      []string{},
			expectedErrors:       []error{},
			delayedStart:         true,
		},

		{
			name: "taskString_dynamic_delayedStart_tasksChannelFull",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 4,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, false, false, false)
			},
			expectedAddTaskError: &errAddTask{i: 5, shoudPanic: true},
			expectedResults:      getExpectedResults(1, 2, 3, 4),
			expectedErrors:       []error{},
			delayedStart:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, testFn(test))
	}
}
