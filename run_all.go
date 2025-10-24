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
	w, done, err := newWorkersForRunAll[R](ctx, tasks, opts...)
	if err != nil {
		return nil, err
	}

	started := enqueueWrappedTasks[R](w, tasks, done)
	waitCompletions(ctx, started, done)

	// Close before draining channels so the ranges terminate cleanly.
	w.Close()
	return collectResultsAndErrors[R](w, len(tasks))
}

// newWorkersForRunAll constructs and starts a Workers instance with a StopOnError buffer sized for the batch.
// It returns the instance and a completion channel sized to the number of tasks.
func newWorkersForRunAll[R any](
	ctx context.Context, tasks []Task[R], opts ...Option,
) (w *Workers[R], done chan struct{}, err error) {
	// Ensure internal StopOnError buffer is large enough to avoid worker send blocking after cancellation.
	opts = append(opts, WithStopOnErrorBuffer(uint(len(tasks))))

	w, err = New[R](ctx, opts...)
	if err != nil {
		return nil, nil, err
	}
	w.Start(ctx)
	done = make(chan struct{}, len(tasks))
	return w, done, nil
}

// enqueueWrappedTasks wraps each task to signal completion and enqueues until AddTask fails.
// It returns the number of tasks that actually started.
func enqueueWrappedTasks[R any](w *Workers[R], tasks []Task[R], done chan struct{}) int {
	started := 0
	for _, t := range tasks {
		taskToRun := t // Capture loop variable.
		var wrappedTask Task[R]
		if taskToRun.SendResult() {
			wrappedTask = TaskFunc[R](func(c context.Context) (R, error) {
				r, e := taskToRun.Run(c)
				done <- struct{}{}
				return r, e
			}).WithID(taskToRun.ID())
		} else {
			wrappedTask = TaskError[R](func(c context.Context) error {
				_, e := taskToRun.Run(c)
				done <- struct{}{}
				return e
			}).WithID(taskToRun.ID())
		}

		if err := w.AddTask(wrappedTask); err != nil {
			break
		}
		started++
	}
	return started
}

// waitCompletions waits for exactly started completion signals.
func waitCompletions(ctx context.Context, started int, done chan struct{}) {
	for i := 0; i < started; i++ {
		select {
		case <-ctx.Done():
			return
		case <-done:
		}
	}
}

// collectResultsAndErrors drains outputs and returns results and aggregated error.
func collectResultsAndErrors[R any](w *Workers[R], capHint int) ([]R, error) {
	results := make([]R, 0, capHint)
	errs := make([]error, 0, capHint)
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
