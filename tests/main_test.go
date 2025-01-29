package tests

import (
	"context"
	"errors"
	"strings"
	"time"
)

type taskResultError = func(context.Context) (string, error)
type taskResult = func(context.Context) string
type taskError = func(context.Context) error

func newTaskResultError(size int, wait time.Duration) taskResultError {
	return func(context.Context) (string, error) {
		s := make([]string, size)
		for i := range s {
			s[i] = strings.Repeat("s", size)
		}

		ss := strings.Join(s, " ")

		time.Sleep(wait)

		return ss, nil
	}
}

func newTaskResult(size int, wait time.Duration) taskResult {
	return func(context.Context) string {
		s := make([]string, size)
		for i := range s {
			s[i] = strings.Repeat("s", size)
		}

		ss := strings.Join(s, " ")

		time.Sleep(wait)

		return ss
	}
}

func newTaskError(size int, wait time.Duration) taskError {
	return func(context.Context) error {
		s := make([]string, size)
		for i := range s {
			s[i] = strings.Repeat("s", size)
		}

		time.Sleep(wait)

		return nil
	}
}

func newErrorTaskResultError(e error) taskResultError {
	return func(context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "", e
	}
}

func newPanicTaskResultError() taskResultError {
	return func(context.Context) (string, error) {
		time.Sleep(100 * time.Millisecond)
		panic("panic")
	}
}

func newPanicTaskResult() taskResult {
	return func(context.Context) string {
		time.Sleep(100 * time.Millisecond)
		panic("panic")
	}
}

func newPanicTaskError(wait time.Duration) taskError {
	return func(context.Context) error {
		time.Sleep(wait)
		panic("panic")
	}
}

func generateTasks(n int, fn taskResultError) []interface{} {
	out := make([]interface{}, n)
	for i := range out {
		out[i] = fn
	}
	return out
}

func generateExpected(n int, fn interface{}) []string {
	out := make([]string, n)
	var s string

	switch typed := fn.(type) {
	case taskResultError:
		s, _ = typed(context.Background())
	case taskResult:
		s = typed(context.Background())
	}

	for i := range out {
		out[i] = s
	}
	return out
}

var (
	tasksN8Size10   = generateTasks(8, newTaskResultError(1024*10, 50*time.Millisecond))
	tasksN256Size2  = generateTasks(256, newTaskResultError(1024*2, 50*time.Millisecond))
	tasksN2048Size2 = generateTasks(2048, newTaskResultError(1024*2, 50*time.Millisecond))

	expectedN2048Size2 = generateExpected(2048, newTaskResultError(1024*2, 0))

	errBasic = errors.New("error")
	errPanic = errors.New("task execution panicked: panic")

	basicTaskResultError = newTaskResultError(5, 200*time.Millisecond)
	basicTaskResult      = newTaskResult(5, 200*time.Millisecond)
	basicTaskError       = newTaskError(5, 200*time.Millisecond)
	longTaskResultError  = newTaskResultError(5, 2*time.Second)
	errorTaskResultError = newErrorTaskResultError(errBasic)
	panicTaskResultError = newPanicTaskResultError()
	panicTaskResult      = newPanicTaskResult()
	panicLongTaskError   = newPanicTaskError(700 * time.Millisecond)
)
