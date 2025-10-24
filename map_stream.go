package workers

import "context"

// MapStream consumes a stream of input items from `in`, applies fn concurrently using the
// Workers engine, and returns the workers' results and errors channels. A non-nil error is
// returned only for immediate setup failures (e.g., invalid options). Runtime errors from
// fn are delivered via the returned errors channel.
//
// Lifecycle:
//   - Constructs a Workers[R] via NewOptions(ctx, opts...) and starts it immediately.
//   - Spawns a forwarder goroutine that reads values from `in`, wraps them into tasks that
//     call fn, and enqueues via AddTask. The forwarder waits for all started tasks to complete
//     before invoking Close(), ensuring results/errors are closed after work is done.
//   - Intake stops when ctx.Done() fires, input is closed, or AddTask returns an error
//     (for example ErrInvalidState after StopOnError cancellation).
//
// Ordering:
//   - Results are emitted in completion order by default; if WithPreserveOrder() is provided,
//     results are emitted in the original input order.
func MapStream[T any, R any](
	ctx context.Context, in <-chan T, fn func(context.Context, T) (R, error), opts ...Option,
) (results <-chan R, errors <-chan error, errOut error) {
	w, err := NewOptions[R](ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	w.Start(ctx)

	go func() {
		defer w.Close()

		done := make(chan struct{}, 1024)
		started := 0
		intake := true

		for intake {
			select {
			case <-ctx.Done():
				intake = false
			case v, ok := <-in:
				if !ok {
					intake = false
					break
				}
				item := v
				orig := TaskFunc[R](func(c context.Context) (R, error) { return fn(c, item) })
				wrapped := TaskFunc[R](func(c context.Context) (R, error) {
					res, e := orig.Run(c)
					done <- struct{}{}
					return res, e
				})
				if err := w.AddTask(wrapped); err != nil {
					intake = false
					break
				}
				started++
			}
		}

		for i := 0; i < started; i++ {
			<-done
		}
	}()

	return w.GetResults(), w.GetErrors(), nil
}
