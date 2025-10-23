package workers

import "context"

// RunStream consumes tasks from the input channel and executes them using a freshly
// constructed Workers instance configured by opts. It returns the workers' results
// and errors channels. A non-nil error is returned only for immediate setup failures
// (e.g., invalid options). Runtime task errors are delivered via the returned errors
// channel.
//
// Lifecycle:
//   - Constructs Workers via NewOptions(ctx, opts...). If successful, starts it immediately.
//   - Spawns a forwarder goroutine that reads tasks from `in`, wraps them to signal completion,
//     and calls AddTask. The forwarder then waits for all started tasks to complete before calling Close().
//     The forwarder stops intake when:
//   - ctx.Done() fires,
//   - the input channel is closed, or
//   - AddTask returns an error (e.g., ErrInvalidState when StopOnError cancels the controller).
//   - Close() waits for in-flight tasks and then closes the results and errors channels.
//
// Ordering:
// - By default, results are emitted in completion order.
// - If WithPreserveOrder() is provided, the core emits results in the original input order.
//
// StopOnError:
//   - If WithStopOnError() is provided, the controller cancels on the first error; the forwarder
//     will observe AddTask returning an error and will stop reading further tasks, then wait for
//     started tasks to signal completion and Close().
//
//nolint:gocritic // ignore unnamed results.
func RunStream[R any](ctx context.Context, in <-chan Task[R], opts ...Option) (<-chan R, <-chan error, error) {
	w, err := NewOptions[R](ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	// Start immediately regardless of WithStartImmediately; Start is idempotent.
	w.Start(ctx)

	// Forwarder that wraps tasks to signal completion; mirrors RunAll's approach
	// to avoid canceling active tasks prematurely when the input closes.
	go func() {
		defer w.Close()

		// completion signaling
		done := make(chan struct{}, 1024) // amortize small bursts; unbounded by waiting below
		started := 0

		// intake loop
		intake := true
		for intake {
			select {
			case <-ctx.Done():
				intake = false
			case t, ok := <-in:
				if !ok {
					intake = false
					break
				}
				// wrap to signal completion regardless of success/error
				orig := t
				var wrapped Task[R]
				if orig.SendResult() {
					wrapped = TaskFunc[R](func(c context.Context) (R, error) {
						res, e := orig.Run(c)
						done <- struct{}{}
						return res, e
					}).WithID(orig.ID())
				} else {
					wrapped = TaskError[R](func(c context.Context) error {
						_, e := orig.Run(c)
						done <- struct{}{}
						return e
					}).WithID(orig.ID())
				}

				if err := w.AddTask(wrapped); err != nil {
					// Typically ErrInvalidState when StopOnError fired; stop intake
					intake = false
					break
				}
				started++
			}
		}

		// wait for completions of all started tasks
		for i := 0; i < started; i++ {
			<-done
		}
	}()

	return w.GetResults(), w.GetErrors(), nil
}
