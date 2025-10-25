package tests

import (
	"context"
	"runtime"
	"testing"

	"github.com/ygrebnov/workers"
)

var errorTaskContextCancelled = "workers: task execution cancelled: " + context.DeadlineExceeded.Error()

func TestCancelContext(t *testing.T) {
	tests := []testCase{
		// immediate start tests.
		{
			name: "taskStringError_dynamic_startImmediately",
			options: []workers.Option{
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, true, false, false))
			},
			// expectedResults: getExpectedResults(1, 2, 3),
			expectedMaxResults: ptrInt(3),
			expectedErrors:     []string{errorTaskContextCancelled, errorTaskContextCancelled},
			twoSetsOfTasks:     true,
			contextTimeout:     true,
		},
		{
			name: "taskStringError_fixed_startImmediately",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, true, false, false))
			},
			// expectedResults: getExpectedResults(1, 2, 3),
			expectedMaxResults: ptrInt(3),
			expectedErrors:     []string{errorTaskContextCancelled, errorTaskContextCancelled},
			twoSetsOfTasks:     true,
			contextTimeout:     true,
		},

		{
			name: "taskString_dynamic_startImmediately",
			options: []workers.Option{
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, true, false, false))
			},
			// expectedResults: getExpectedResults(1, 2, 3),
			expectedMaxResults: ptrInt(3),
			expectedErrors:     []string{errorTaskContextCancelled, errorTaskContextCancelled},
			twoSetsOfTasks:     true,
			contextTimeout:     true,
		},
		{
			name: "taskString_fixed_startImmediately",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, true, false, false))
			},
			// expectedResults: getExpectedResults(1, 2, 3),
			expectedMaxResults: ptrInt(3),
			expectedErrors:     []string{errorTaskContextCancelled, errorTaskContextCancelled},
			twoSetsOfTasks:     true,
			contextTimeout:     true,
		},

		{
			name: "taskError_dynamic_startImmediately",
			options: []workers.Option{
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, true, false, false))
			},
			expectedResults: []string{},
			expectedErrors:  []string{errorTaskContextCancelled, errorTaskContextCancelled},
			twoSetsOfTasks:  true,
			contextTimeout:  true,
		},
		{
			name: "taskError_fixed_startImmediately",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, true, false, false))
			},
			expectedResults: []string{},
			expectedErrors:  []string{errorTaskContextCancelled, errorTaskContextCancelled},
			twoSetsOfTasks:  true,
			contextTimeout:  true,
		},

		// delayed start tests.
		{
			name: "taskStringError_dynamic_delayedStart",
			options: []workers.Option{
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, true, false, false))
			},
			// expectedResults: getExpectedResults(1, 2, 3),
			expectedMaxResults: ptrInt(3),
			expectedErrors:     []string{errorTaskContextCancelled, errorTaskContextCancelled},
			contextTimeout:     true,
			delayedStart:       true,
			twoSetsOfTasks:     true,
		},
		{
			name: "taskStringError_fixed_delayedStart",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskFunc[string](newTaskStringError(i, true, false, false))
			},
			// expectedResults: getExpectedResults(1, 2, 3),
			expectedMaxResults: ptrInt(3),
			expectedErrors:     []string{errorTaskContextCancelled, errorTaskContextCancelled},
			contextTimeout:     true,
			delayedStart:       true,
			twoSetsOfTasks:     true,
		},

		{
			name: "taskString_dynamic_delayedStart",
			options: []workers.Option{
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, true, false, false))
			},
			// expectedResults: getExpectedResults(1, 2, 3),
			expectedMaxResults: ptrInt(3),
			expectedErrors:     []string{errorTaskContextCancelled, errorTaskContextCancelled},
			contextTimeout:     true,
			delayedStart:       true,
			twoSetsOfTasks:     true,
		},
		{
			name: "taskString_fixed_delayedStart",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskValue[string](newTaskString(i, true, false, false))
			},
			// expectedResults: getExpectedResults(1, 2, 3),
			expectedMaxResults: ptrInt(3),
			expectedErrors:     []string{errorTaskContextCancelled, errorTaskContextCancelled},
			contextTimeout:     true,
			delayedStart:       true,
			twoSetsOfTasks:     true,
		},

		{
			name: "taskError_dynamic_delayedStart",
			options: []workers.Option{
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, true, false, false))
			},
			expectedResults: []string{},
			expectedErrors:  []string{errorTaskContextCancelled, errorTaskContextCancelled},
			contextTimeout:  true,
			delayedStart:    true,
			twoSetsOfTasks:  true,
		},
		{
			name: "taskError_fixed_delayedStart",
			options: []workers.Option{
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithTasksBuffer(5), // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) workers.Task[string] {
				return workers.TaskError[string](newTaskErr(i, true, false, false))
			},
			expectedResults: []string{},
			expectedErrors:  []string{errorTaskContextCancelled, errorTaskContextCancelled},
			contextTimeout:  true,
			delayedStart:    true,
			twoSetsOfTasks:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, testFn(&test))
	}
}
