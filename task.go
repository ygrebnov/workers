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

// execTask centralizes goroutine launch, panic recovery, and ctx cancellation.
// The `call` closure must do the actual work and return (R, error).
func execTask[R interface{}](ctx context.Context, call func() (R, error)) (R, error) {
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
			done <- struct{}{}
		}()

		result, err = call()
	}()

	select {
	case <-ctx.Done():
		return *(new(R)), fmt.Errorf("task execution cancelled: %w", ctx.Err())
	case <-done:
		return result, err
	}
}

type taskResultError[R interface{}] struct {
	fn func(ctx context.Context) (R, error)
}

func (t *taskResultError[R]) execute(ctx context.Context) (R, error) {
	return execTask[R](ctx, func() (R, error) { return t.fn(ctx) })
}

type taskResult[R interface{}] struct {
	fn func(ctx context.Context) R
}

func (t *taskResult[R]) execute(ctx context.Context) (R, error) {
	return execTask[R](ctx, func() (R, error) { return t.fn(ctx), nil })
}

type taskError[R interface{}] struct {
	fn func(ctx context.Context) error
}

func (t *taskError[R]) execute(ctx context.Context) (R, error) {
	return execTask[R](ctx, func() (R, error) { var zero R; return zero, t.fn(ctx) })
}
