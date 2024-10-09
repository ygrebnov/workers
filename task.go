package workers

import (
	"context"
	"errors"
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
	return t.fn(ctx)
}

type taskResult[R interface{}] struct {
	fn func(ctx context.Context) R
}

func (t *taskResult[R]) execute(ctx context.Context) (R, error) {
	return t.fn(ctx), nil
}

type taskError[R interface{}] struct {
	fn func(ctx context.Context) error
}

func (t *taskError[R]) execute(ctx context.Context) (R, error) {
	return *(new(R)), t.fn(ctx)
}
