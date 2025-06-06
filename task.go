package workers

import (
	"context"
	"errors"
	"fmt"
)

var ErrInvalidTaskType = errors.New("invalid task type")

type task[R interface{}] interface {
	execute(ctx context.Context) (R, error)
}

func newTask[R interface{}](fn interface{}) (task[R], error) {
	switch typed := fn.(type) {
	case func(context.Context) (R, error):
		return &taskResultError[R]{fn: typed}, nil

	case func(ctx context.Context) R:
		return &taskResult[R]{fn: typed}, nil

	case func(context.Context) error:
		return &taskError[R]{fn: typed}, nil

	default:
		return nil, ErrInvalidTaskType
	}
}

type taskResultError[R interface{}] struct {
	fn func(ctx context.Context) (R, error)
}

func (t *taskResultError[R]) execute(ctx context.Context) (R, error) {
	var (
		result R
		err    error
	)

	done := make(chan struct{}, 1)

	go func() {
		defer func() {
			if ePanic := recover(); ePanic != nil {
				err = fmt.Errorf("task execution panicked: %v", ePanic)
				done <- struct{}{}
			}
		}()

		result, err = t.fn(ctx)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return *(new(R)), fmt.Errorf("task execution cancelled: %w", ctx.Err())
	case <-done:
		return result, err
	}
}

type taskResult[R interface{}] struct {
	fn func(ctx context.Context) R
}

func (t *taskResult[R]) execute(ctx context.Context) (R, error) {
	var (
		result R
		err    error
	)

	done := make(chan struct{}, 1)

	go func() {
		defer func() {
			if ePanic := recover(); ePanic != nil {
				err = fmt.Errorf("task execution panicked: %v", ePanic)
				done <- struct{}{}
			}
		}()

		result = t.fn(ctx)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return *(new(R)), fmt.Errorf("task execution cancelled: %w", ctx.Err())
	case <-done:
		return result, err
	}
}

type taskError[R interface{}] struct {
	fn func(ctx context.Context) error
}

func (t *taskError[R]) execute(ctx context.Context) (R, error) {
	var err error

	done := make(chan struct{}, 1)

	go func() {
		defer func() {
			if ePanic := recover(); ePanic != nil {
				err = fmt.Errorf("task execution panicked: %v", ePanic)
				done <- struct{}{}
			}
		}()

		err = t.fn(ctx)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return *(new(R)), fmt.Errorf("task execution cancelled: %w", ctx.Err())
	case <-done:
		return *(new(R)), err
	}
}
