package workers

import (
	"context"
	"fmt"
)

// Task is the canonical task shape used throughout the package.
// It takes a context and returns a result of type R and an error.
// Use TaskFunc / TaskValue / TaskError helpers to adapt common function signatures.
// Note: R uses interface{} for compatibility with existing generics constraints in this module.
//
//	Future versions may switch to `any`.
//
// Example:
//
//	t := TaskFunc(func(ctx context.Context) (int, error) { return 42, nil })
//	_ = t
//
// The canonical representation simplifies internal execution and public helper APIs.
// It is intentionally a function type to keep overhead minimal.
//
//nolint:revive // generic constraint uses interface{} for compatibility.
type Task[R interface{}] func(context.Context) (R, error)

// TaskFunc adapts func(ctx) (R, error) to Task[R].
func TaskFunc[R interface{}](fn func(context.Context) (R, error)) Task[R] { return Task[R](fn) }

// TaskValue adapts func(ctx) R to Task[R].
func TaskValue[R interface{}](fn func(context.Context) R) Task[R] {
	return func(ctx context.Context) (R, error) { return fn(ctx), nil }
}

// TaskError adapts func(ctx) error to Task[R].
// The returned Task yields the zero value of R alongside the error.
func TaskError[R interface{}](fn func(context.Context) error) Task[R] {
	return func(ctx context.Context) (R, error) { var zero R; return zero, fn(ctx) }
}

// internal task interface used by the workers controller/worker engine.
// Concrete implementations may carry metadata like whether a result should be forwarded.
//
//nolint:revive // generic constraint uses interface{} for compatibility.
type task[R interface{}] interface {
	execute(ctx context.Context) (R, error)
	// sendResult indicates whether a successful execution should be forwarded to results.
	sendResult() bool
}

// taskAdapter adapts a canonical Task[R] into an internal task with a sendResult flag.
type taskAdapter[R interface{}] struct {
	fn          Task[R]
	_sendResult bool
}

func (t *taskAdapter[R]) execute(ctx context.Context) (R, error) {
	return execTask[R](ctx, func() (R, error) { return t.fn(ctx) })
}

func (t *taskAdapter[R]) sendResult() bool { return t._sendResult }

// newTask converts supported inputs into the internal task[R] using the canonical Task[R].
func newTask[R interface{}](fn interface{}) (task[R], error) {
	switch typed := fn.(type) {
	case func(context.Context) (R, error):
		return &taskAdapter[R]{fn: TaskFunc[R](typed), _sendResult: true}, nil

	case func(context.Context) R:
		return &taskAdapter[R]{fn: TaskValue[R](typed), _sendResult: true}, nil

	case func(context.Context) error:
		return &taskAdapter[R]{fn: TaskError[R](typed), _sendResult: false}, nil

	case Task[R]:
		return &taskAdapter[R]{fn: typed, _sendResult: true}, nil

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
				err = fmt.Errorf("%w: %v", ErrTaskPanicked, ePanic)
			}
			done <- struct{}{}
		}()

		result, err = call()
	}()

	select {
	case <-ctx.Done():
		return *(new(R)), fmt.Errorf("%w: %w", ErrTaskCancelled, ctx.Err())
	case <-done:
		return result, err
	}
}
