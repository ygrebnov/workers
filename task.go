package workers

import (
	"context"
	"errors"
	"fmt"
)

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
		return nil, errors.New("invalid task type")
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
			}
		}()

		result, err = t.fn(ctx)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return *(new(R)), ctx.Err()
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
			}
		}()

		result = t.fn(ctx)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return *(new(R)), ctx.Err()
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
			}
		}()

		err = t.fn(ctx)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return *(new(R)), ctx.Err()
	case <-done:
		return *(new(R)), err
	}
}
