package workers

import (
	"context"
	"errors"
)

// RunAll executes the provided tasks using a new Workers instance configured by opts.
// It owns the lifecycle: Start, enqueue all tasks, wait for started tasks to finish, Close, then collect outputs.
//
// Semantics:
// - Results are returned in completion order (not input order).
// - If StopOnError is enabled in opts, cancellation is triggered on the first error; some tasks may not start.
// - The returned error is errors.Join of all task errors (nil if no errors).
func RunAll[R any](ctx context.Context, tasks []Task[R], opts ...Option) ([]R, error) {
	// Ensure internal StopOnError buffer is large enough to avoid worker send blocking after cancellation.
	opts = append(opts, WithStopOnErrorBuffer(uint(len(tasks))))

	w, err := NewOptions[R](ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Always start explicitly; Start is idempotent if WithStartImmediately was set.
	w.Start(ctx)

	// Completion channel to await only tasks that actually got enqueued.
	done := make(chan struct{}, len(tasks))

	// Wrap tasks so each completion signals done.
	wrap := func(t Task[R]) Task[R] {
		if t.SendResult() {
			return TaskFunc[R](func(c context.Context) (R, error) {
				r, e := t.Run(c)
				done <- struct{}{}
				return r, e
			})
		}
		return TaskError[R](func(c context.Context) error {
			_, e := t.Run(c)
			done <- struct{}{}
			return e
		})
	}

	// Enqueue wrapped tasks until AddTask fails (e.g., cancellation due to StopOnError).
	started := 0
	for _, t := range tasks {
		if err := w.AddTask(wrap(t)); err != nil {
			break
		}
		started++
	}

	// Wait for all started tasks to signal completion.
	for i := 0; i < started; i++ {
		select {
		case <-done:
			// ok
		case <-ctx.Done():
			// Context canceled by caller; continue to Close and drain.
		}
	}

	// Now it's safe to Close: no in-flight tasks remain.
	w.Close()

	// Drain channels.
	var (
		results []R
		errs    []error
	)
	for r := range w.GetResults() {
		results = append(results, r)
	}
	for e := range w.GetErrors() {
		if e != nil {
			errs = append(errs, e)
		}
	}

	return results, errors.Join(errs...)
}
