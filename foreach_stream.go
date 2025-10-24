package workers

import "context"

// ForEachStream applies fn to each item from the input channel concurrently using the Workers engine.
// It returns an errors channel carrying per-item failures and a setup error for immediate issues only.
// The returned errors channel is closed when the stream is fully processed or canceled.
//
// Lifecycle:
//   - Constructs Workers via NewOptions(ctx, opts...), starts it immediately.
//   - Spawns a forwarder goroutine that reads from `in`, wraps each item into a TaskError,
//     and calls AddTask. It waits for all started tasks to complete, then calls Close().
//   - Intake stops on ctx.Done(), input channel close, or AddTask error
//     (e.g., ErrInvalidState when StopOnError is enabled and the first error occurred).
func ForEachStream[T any](
	ctx context.Context, in <-chan T, fn func(context.Context, T) error, opts ...Option,
) (<-chan error, error) {
	w, err := NewOptions[struct{}](ctx, opts...)
	if err != nil {
		return nil, err
	}

	w.Start(ctx)

	go func() {
		defer w.Close()

		// TODO: make this buffered channel size configurable via options
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
				// Build the original error-only task.
				orig := TaskError[struct{}](func(c context.Context) error { return fn(c, item) })
				// Wrap to signal completion after Run(c) returns (even on cancellation).
				wrapped := TaskError[struct{}](func(c context.Context) error { // TODO: simplify double-wrapping
					_, e := orig.Run(c)
					done <- struct{}{}
					return e
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

	return w.GetErrors(), nil
}
