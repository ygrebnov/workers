package workers

import (
	"context"
	"fmt"
)

// Task encapsulates a unit of work function and whether a successful execution
// should emit a result to the results channel.
// Use TaskFunc / TaskValue / TaskError to construct instances.
type Task[R interface{}] struct {
	fn          func(context.Context) (R, error)
	_sendResult bool
}

// TaskFunc adapts func(ctx) (R, error) to a Task that emits results on success.
func TaskFunc[R interface{}](fn func(context.Context) (R, error)) Task[R] {
	return Task[R]{fn: fn, _sendResult: true}
}

// TaskValue adapts func(ctx) R to a Task that emits results on success.
func TaskValue[R interface{}](fn func(context.Context) R) Task[R] {
	return Task[R]{fn: func(ctx context.Context) (R, error) { return fn(ctx), nil }, _sendResult: true}
}

// TaskError adapts func(ctx) error to a Task that does NOT emit results.
func TaskError[R interface{}](fn func(context.Context) error) Task[R] {
	return Task[R]{fn: func(ctx context.Context) (R, error) { var zero R; return zero, fn(ctx) }, _sendResult: false}
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

// Run executes the task function with panic recovery and context cancellation.
func (t Task[R]) Run(ctx context.Context) (R, error) {
	return execTask[R](ctx, func() (R, error) { return t.fn(ctx) })
}

// SendResult reports whether a successful run should be forwarded to results.
func (t Task[R]) SendResult() bool { return t._sendResult }
