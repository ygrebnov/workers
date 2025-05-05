package tests

import (
	"runtime"
	"testing"

	"github.com/ygrebnov/workers"
)

func TestHandlePanic(t *testing.T) {
	tests := []testCase{
		// start immediately, do not stop on error.
		{
			name: "taskStringError_dynamic_startImmediately",
			config: &workers.Config{
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
		},
		{
			name: "taskStringError_fixed_startImmediately",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
		},

		{
			name: "taskString_dynamic_startImmediately",
			config: &workers.Config{
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
		},
		{
			name: "taskString_fixed_startImmediately",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
		},

		{
			name: "taskError_dynamic_startImmediately",
			config: &workers.Config{
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, true)
			},
			expectedResults: []string{},
			expectedErrors: []string{
				"task execution panicked: panic on executing task for: 3",
			},
		},
		{
			name: "taskError_fixed_startImmediately",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, true)
			},
			expectedResults: []string{},
			expectedErrors: []string{
				"task execution panicked: panic on executing task for: 3",
			},
		},

		// delayed start, do not stop on error.
		{
			name: "taskStringError_dynamic_delayedStart",
			config: &workers.Config{
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors: []string{
				"task execution panicked: panic on executing task for: 3",
			},
			delayedStart: true,
		},
		{
			name: "taskStringError_fixed_delayedStart",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors: []string{
				"task execution panicked: panic on executing task for: 3",
			},
			delayedStart: true,
		},

		{
			name: "taskString_dynamic_delayedStart",
			config: &workers.Config{
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
			delayedStart: true,
		},
		{
			name: "taskString_fixed_delayedStart",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
			delayedStart: true,
		},

		{
			name: "taskError_dynamic_delayedStart",
			config: &workers.Config{
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, true)
			},
			expectedResults: []string{},
			expectedErrors: []string{
				"task execution panicked: panic on executing task for: 3",
			},
			delayedStart: true,
		},
		{
			name: "taskError_fixed_delayedStart",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, true)
			},
			expectedResults: []string{},
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
			delayedStart:    true,
		},

		// start immediately, stop on error.
		{
			name: "taskStringError_dynamic_startImmediately_stopOnError",
			config: &workers.Config{
				StartImmediately: true,
				StopOnError:      true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2),
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
		},
		{
			name: "taskStringError_fixed_startImmediately_stopOnError",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
				StopOnError:      true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2),
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
		},

		{
			name: "taskString_dynamic_startImmediately_stopOnError",
			config: &workers.Config{
				StartImmediately: true,
				StopOnError:      true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
		},
		{
			name: "taskString_fixed_startImmediately_stopOnError",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
				StopOnError:      true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
		},

		{
			name: "taskError_dynamic_startImmediately_stopOnError",
			config: &workers.Config{
				StartImmediately: true,
				StopOnError:      true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, true)
			},
			expectedResults: []string{},
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
		},
		{
			name: "taskError_fixed_startImmediately_stopOnError",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
				StopOnError:      true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, true)
			},
			expectedResults: []string{},
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
		},

		// delayed start, stop on error.
		{
			name: "taskStringError_dynamic_delayedStart_stopOnError",
			config: &workers.Config{
				TasksBufferSize: 5, // the size is the same as the number of tasks.
				StopOnError:     true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2),
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
			delayedStart:    true,
		},
		{
			name: "taskStringError_fixed_delayedStart_stopOnError",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
				StopOnError:     true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2),
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
			delayedStart:    true,
		},

		{
			name: "taskString_dynamic_delayedStart_stopOnError",
			config: &workers.Config{
				TasksBufferSize: 5, // the size is the same as the number of tasks.
				StopOnError:     true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
			delayedStart: true,
		},
		{
			name: "taskString_fixed_delayedStart_stopOnError",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
				StopOnError:     true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, true)
			},
			expectedResults: getExpectedResults(1, 2),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
			delayedStart: true,
		},

		{
			name: "taskError_dynamic_delayedStart_stopOnError",
			config: &workers.Config{
				TasksBufferSize: 5, // the size is the same as the number of tasks.
				StopOnError:     true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, true)
			},
			expectedResults: []string{},
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
			delayedStart:    true,
		},
		{
			name: "taskError_fixed_delayedStart_stopOnError",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
				StopOnError:     true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, true)
			},
			expectedResults: []string{},
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
			delayedStart:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, testFn(&test))
	}
}
