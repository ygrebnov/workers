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
			config: &workers.Config{
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, false, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []error{},
		},

		{
			name: "taskStringError_fixed_startImmediately",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, false, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []error{},
		},

		{
			name: "taskString_dynamic_startImmediately",
			config: &workers.Config{
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, false, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []error{},
		},

		{
			name: "taskString_fixed_startImmediately",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, false, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []error{},
		},

		{
			name: "taskError_dynamic_startImmediately",
			config: &workers.Config{
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, false, false, false)
			},
			expectedResults: []string{},
			expectedErrors:  []error{},
		},

		{
			name: "taskError_fixed_startImmediately",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, false, false, false)
			},
			expectedResults: []string{},
			expectedErrors:  []error{},
		},

		// delayed start tests.
		{
			name: "taskStringError_dynamic_delayedStart",
			config: &workers.Config{
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, false, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []error{},
			delayedStart:    true,
		},

		{
			name: "taskStringError_fixed_delayedStart",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, false, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []error{},
			delayedStart:    true,
		},

		{
			name: "taskString_dynamic_delayedStart",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, false, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []error{},
			delayedStart:    true,
		},

		{
			name: "taskString_fixed_delayedStart",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, false, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3, 4, 5),
			expectedErrors:  []error{},
			delayedStart:    true,
		},

		{
			name: "taskError_dynamic_delayedStart",
			config: &workers.Config{
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, false, false, false)
			},
			expectedResults: []string{},
			expectedErrors:  []error{},
			delayedStart:    true,
		},
		{
			name: "taskError_fixed_delayedStart",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, false, false, false)
			},
			expectedResults: []string{},
			expectedErrors:  []error{},
			delayedStart:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, testFn(&test))
	}
}
