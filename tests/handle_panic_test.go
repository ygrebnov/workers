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
			options: []workers.Option{
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, true, false, true))
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
		},
		{
			name: "taskStringError_fixed_startImmediately",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, true, false, true))
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
		},

		{
			name: "taskString_dynamic_startImmediately",
			options: []workers.Option{
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, true, false, true))
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
		},
		{
			name: "taskString_fixed_startImmediately",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, true, false, true))
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
		},

		{
			name: "taskError_dynamic_startImmediately",
			options: []workers.Option{
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, true, false, true))
			},
			expectedResults: []string{},
			expectedErrors: []string{
				"task execution panicked: panic on executing task for: 3",
			},
		},
		{
			name: "taskError_fixed_startImmediately",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, true, false, true))
			},
			expectedResults: []string{},
			expectedErrors: []string{
				"task execution panicked: panic on executing task for: 3",
			},
		},

		// delayed start, do not stop on error.
		{
			name: "taskStringError_dynamic_delayedStart",
			options: []workers.Option{
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, true, false, true))
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors: []string{
				"task execution panicked: panic on executing task for: 3",
			},
			delayedStart: true,
		},
		{
			name: "taskStringError_fixed_delayedStart",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, true, false, true))
			},
			expectedResults: getExpectedResults(1, 2, 4, 5),
			expectedErrors: []string{
				"task execution panicked: panic on executing task for: 3",
			},
			delayedStart: true,
		},

		{
			name: "taskString_dynamic_delayedStart",
			options: []workers.Option{
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, true, false, true))
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
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, true, false, true))
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
			options: []workers.Option{
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, true, false, true))
			},
			expectedResults: []string{},
			expectedErrors: []string{
				"task execution panicked: panic on executing task for: 3",
			},
			delayedStart: true,
		},
		{
			name: "taskError_fixed_delayedStart",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, true, false, true))
			},
			expectedResults: []string{},
			expectedErrors:  []string{"task execution panicked: panic on executing task for: 3"},
			delayedStart:    true,
		},

		// start immediately, stop on error.
		{
			name: "taskStringError_dynamic_startImmediately_stopOnError",
			options: []workers.Option{
				workers.WithStartImmediately(),
				workers.WithStopOnError(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, true, false, true))
			},
			// expectedResults: getExpectedResults(1, 2),
			expectedMaxResults: ptrInt(4),
			expectedErrors:     []string{"task execution panicked: panic on executing task for: 3"},
		},
		{
			name: "taskStringError_fixed_startImmediately_stopOnError",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
				workers.WithStopOnError(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, true, false, true))
			},
			// expectedResults: getExpectedResults(1, 2),
			expectedMaxResults: ptrInt(4),
			expectedErrors:     []string{"task execution panicked: panic on executing task for: 3"},
		},

		{
			name: "taskString_dynamic_startImmediately_stopOnError",
			options: []workers.Option{
				workers.WithStartImmediately(),
				workers.WithStopOnError(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, true, false, true))
			},
			// expectedResults: getExpectedResults(1, 2),
			expectedMaxResults: ptrInt(4),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
		},
		{
			name: "taskString_fixed_startImmediately_stopOnError",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
				workers.WithStopOnError(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, true, false, true))
			},
			// expectedResults: getExpectedResults(1, 2),
			expectedMaxResults: ptrInt(4),
			expectedErrors: []string{
				// an error is sent through the channel because panic is handled at one level higher than the task.
				"task execution panicked: panic on executing task for: 3",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, testFn(&test))
	}
}
