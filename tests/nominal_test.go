package tests

import (
	"runtime"
	"testing"

	"github.com/ygrebnov/workers"
)

func TestNominal(t *testing.T) {
	tests := []testCase{
		// immediate start tests.
		{
			name: "taskStringError_dynamic_startImmediately",
			options: []workers.Option{
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, false, false, false))
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
		},

		{
			name: "taskStringError_fixed_startImmediately",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, false, false, false))
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
		},

		{
			name: "taskString_dynamic_startImmediately",
			options: []workers.Option{
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, false, false, false))
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
		},

		{
			name: "taskString_fixed_startImmediately",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, false, false, false))
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
		},

		{
			name: "taskError_dynamic_startImmediately",
			options: []workers.Option{
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, false, false, false))
			},
			expectedResults: []string{},
		},

		{
			name: "taskError_fixed_startImmediately",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, false, false, false))
			},
			expectedResults: []string{},
		},

		// delayed start tests.
		{
			name: "taskStringError_dynamic_delayedStart",
			options: []workers.Option{
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, false, false, false))
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			delayedStart:    true,
		},

		{
			name: "taskStringError_fixed_delayedStart",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, false, false, false))
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			delayedStart:    true,
		},

		{
			name: "taskString_dynamic_delayedStart",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, false, false, false))
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			delayedStart:    true,
		},

		{
			name: "taskString_fixed_delayedStart",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, false, false, false))
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			delayedStart:    true,
		},

		{
			name: "taskError_dynamic_delayedStart",
			options: []workers.Option{
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, false, false, false))
			},
			expectedResults: []string{},
			delayedStart:    true,
		},
		{
			name: "taskError_fixed_delayedStart",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, false, false, false))
			},
			expectedResults: []string{},
			delayedStart:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, testFn(&test))
	}
}
