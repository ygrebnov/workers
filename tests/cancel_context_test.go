package tests

import (
	"context"
	"runtime"
	"testing"

	"github.com/ygrebnov/workers"
)

func TestCancelContext(t *testing.T) {
	tests := []testCase{
		// immediate start tests.
		{
			name: "taskStringError_dynamic_startImmediately",
			config: &workers.Config{
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3),
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			twoSetsOfTasks:  true,
			contextTimeout:  true,
		},
		{
			name: "taskStringError_fixed_startImmediately",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3),
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			twoSetsOfTasks:  true,
			contextTimeout:  true,
		},

		{
			name: "taskString_dynamic_startImmediately",
			config: &workers.Config{
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3),
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			twoSetsOfTasks:  true,
			contextTimeout:  true,
		},
		{
			name: "taskString_fixed_startImmediately",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3),
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			twoSetsOfTasks:  true,
			contextTimeout:  true,
		},

		{
			name: "taskError_dynamic_startImmediately",
			config: &workers.Config{
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, false)
			},
			expectedResults: []string{},
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			twoSetsOfTasks:  true,
			contextTimeout:  true,
		},
		{
			name: "taskError_fixed_startImmediately",
			config: &workers.Config{
				MaxWorkers:       uint(runtime.NumCPU()),
				StartImmediately: true,
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, false)
			},
			expectedResults: []string{},
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			twoSetsOfTasks:  true,
			contextTimeout:  true,
		},

		// delayed start tests.
		{
			name: "taskStringError_dynamic_delayedStart",
			config: &workers.Config{
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3),
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			contextTimeout:  true,
			delayedStart:    true,
			twoSetsOfTasks:  true,
		},
		{
			name: "taskStringError_fixed_delayedStart",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskStringError(i, true, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3),
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			contextTimeout:  true,
			delayedStart:    true,
			twoSetsOfTasks:  true,
		},

		{
			name: "taskString_dynamic_delayedStart",
			config: &workers.Config{
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3),
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			contextTimeout:  true,
			delayedStart:    true,
			twoSetsOfTasks:  true,
		},
		{
			name: "taskString_fixed_delayedStart",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskString(i, true, false, false)
			},
			expectedResults: getExpectedResults(1, 2, 3),
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			contextTimeout:  true,
			delayedStart:    true,
			twoSetsOfTasks:  true,
		},

		{
			name: "taskError_dynamic_delayedStart",
			config: &workers.Config{
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, false)
			},
			expectedResults: []string{},
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			contextTimeout:  true,
			delayedStart:    true,
			twoSetsOfTasks:  true,
		},
		{
			name: "taskError_fixed_delayedStart",
			config: &workers.Config{
				MaxWorkers:      uint(runtime.NumCPU()),
				TasksBufferSize: 5, // the size is the same as the number of tasks.
			},
			nTasks: 5,
			task: func(i int) interface{} {
				return newTaskErr(i, true, false, false)
			},
			expectedResults: []string{},
			expectedErrors:  []error{context.DeadlineExceeded, context.DeadlineExceeded},
			contextTimeout:  true,
			delayedStart:    true,
			twoSetsOfTasks:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, testFn(&test))
	}
}
