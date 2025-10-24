package workers

import (
	"context"
	"fmt"
)

// Task encapsulates a unit of work function and whether a successful execution
// should emit a result to the results channel.
// Use TaskFunc / TaskValue / TaskError to construct instances.
//
// Task carries an optional opaque identifier set by constructors with ID
// or via WithID. The ID is intended for correlation and observability.
type Task[R interface{}] struct {
	fn          func(context.Context) (R, error)
	_sendResult bool
	_id         any
	_index      int
}

// TaskFunc adapts func(ctx) (R, error) to a Task that emits results on success.
func TaskFunc[R interface{}](fn func(context.Context) (R, error)) Task[R] {
	return Task[R]{fn: fn, _sendResult: true}
}

// TaskFuncWithID adapts func(ctx) (R, error) to a Task with a provided ID that emits results on success.
func TaskFuncWithID[R interface{}](id any, fn func(context.Context) (R, error)) Task[R] {
	return Task[R]{fn: fn, _sendResult: true, _id: id}
}

// TaskValue adapts func(ctx) R to a Task that emits results on success.
func TaskValue[R interface{}](fn func(context.Context) R) Task[R] {
	return Task[R]{fn: func(ctx context.Context) (R, error) { return fn(ctx), nil }, _sendResult: true}
}

// TaskValueWithID adapts func(ctx) R to a Task with a provided ID that emits results on success.
func TaskValueWithID[R interface{}](id any, fn func(context.Context) R) Task[R] {
	return Task[R]{fn: func(ctx context.Context) (R, error) { return fn(ctx), nil }, _sendResult: true, _id: id}
}

// TaskError adapts func(ctx) error to a Task that does NOT emit results.
func TaskError[R interface{}](fn func(context.Context) error) Task[R] {
	return Task[R]{fn: func(ctx context.Context) (R, error) { var zero R; return zero, fn(ctx) }, _sendResult: false}
}

// TaskErrorWithID adapts func(ctx) error to a Task with a provided ID that does NOT emit results.
func TaskErrorWithID[R interface{}](id any, fn func(context.Context) error) Task[R] {
	return Task[R]{
		fn:          func(ctx context.Context) (R, error) { var zero R; return zero, fn(ctx) },
		_sendResult: false,
		_id:         id,
	}
}

// WithID returns a shallow copy of t with its ID set to id.
func (t Task[R]) WithID(id any) Task[R] { t._id = id; return t }

// WithIndex returns a shallow copy of t with its input sequence index set.
// This is used internally for error correlation when tagging is enabled.
func (t Task[R]) WithIndex(i int) Task[R] { t._index = i; return t }

// ID returns the opaque identifier associated with the task (may be nil).
func (t Task[R]) ID() any { return t._id }

// Index returns the input sequence index if present (true when set by Workers).
func (t Task[R]) Index() (int, bool) { return t._index, t._index != 0 || t._id != nil }

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
