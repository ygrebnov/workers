package workers

import (
	"context"
	"sync"

	"github.com/ygrebnov/workers/pool"
)

// Workers manages a pool of workers executing typed tasks and exposing results/errors channels.
// Workers is a concrete struct; methods are safe for concurrent use.
// Zero-value is usable: call Start(ctx) to initialize with defaults (or construct via New/NewOptions).
type Workers[R interface{}] struct {
	// noCopy prevents accidental copying of the controller.
	//go:nocopy
	nc noCopy

	config *Config

	once      sync.Once
	closeOnce sync.Once

	// internal lifecycle control
	ctx    context.Context
	cancel context.CancelFunc

	// worker pool
	pool pool.Pool

	// channels
	tasks   chan Task[R]
	results chan R
	errors  chan error // outward errors channel

	// When StopOnError is enabled, workers produce into this smaller internal buffer,
	// which Start() drains and forwards into the outward errors channel, then cancels.
	errorsBuf chan error

	// in-flight tasks accounting (dispatch wrappers increment/decrement)
	inflight sync.WaitGroup

	// waits for the stop-on-error forwarder goroutine to exit before closing errors
	forwarderWG sync.WaitGroup

	// tracks detached sender goroutines that forward outward errors asynchronously
	errorsSendWG sync.WaitGroup

	// closed during Close to unblock any pending detached senders safely
	closeCh chan struct{}
}

// noCopy is a vet-recognized marker to discourage copying types with this field embedded.
// It works with the "-copylocks" analyzer via the presence of Lock/Unlock methods.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// New creates a new Workers object instance and returns it.
//
// Deprecated: This Config-based constructor will be deprecated in a future release.
// Prefer NewOptions(ctx, opts...) which will become the primary New in the next major version.
//
// The Workers object is not started automatically.
// To start it, either 'StartImmediately' configuration option must be set to true or
// the Start method must be called explicitly.
func New[R interface{}](ctx context.Context, config *Config) *Workers[R] {
	if config == nil {
		cfg := defaultConfig()
		config = &cfg
	}

	if err := validateConfig(config); err != nil {
		panic(err)
	}

	r := make(chan R, config.ResultsBufferSize)

	// Prepare the channel that workers will write errors to.
	// In StopOnError mode, workers produce into a smaller internal buffer (errorsBuf)
	// which the controller drains and forwards to the outward errors channel.
	var workerErrors chan error
	if config.StopOnError {
		workerErrors = make(chan error, config.StopOnErrorErrorsBufferSize)
	} else {
		workerErrors = make(chan error, config.ErrorsBufferSize)
	}

	newWorkerFn := func() interface{} { return newWorker(r, workerErrors) }

	var p pool.Pool
	if config.MaxWorkers > 0 {
		p = pool.NewFixed(config.MaxWorkers, newWorkerFn)
	} else {
		p = pool.NewDynamic(newWorkerFn)
	}

	tasks := make(chan Task[R], config.TasksBufferSize)
	if config.TasksBufferSize == 0 {
		tasks = nil // to return error in AddTask.
	}

	w := &Workers[R]{
		config:  config,
		tasks:   tasks,
		results: r,
		pool:    p,
	}

	if config.StopOnError {
		// outward errors channel keeps a larger buffer for receivers
		w.errors = make(chan error, config.ErrorsBufferSize)
		w.errorsBuf = workerErrors
	} else {
		// in non-stoppable mode, workers write directly to the outward errors channel
		w.errors = workerErrors
	}

	if config.StartImmediately {
		w.Start(ctx)
	}

	return w
}

// Start starts the Workers and begins executing tasks.
func (w *Workers[R]) Start(ctx context.Context) {
	w.once.Do(func() {
		w.initDefaultsIfNeeded()
		w.initChannelsIfNeeded()
		w.initPoolIfNeeded()
		w.initContext(ctx)
		w.startErrorForwarderIfNeeded()
		w.startDispatcher(w.ctx)
	})
}

// initDefaultsIfNeeded ensures config exists.
func (w *Workers[R]) initDefaultsIfNeeded() {
	if w.config == nil {
		cfg := defaultConfig()
		w.config = &cfg
	}
}

// initChannelsIfNeeded initializes results/errors/errorsBuf/tasks channels as needed.
func (w *Workers[R]) initChannelsIfNeeded() {
	if w.results == nil {
		w.results = make(chan R, w.config.ResultsBufferSize)
	}
	if w.errors == nil {
		w.errors = make(chan error, w.config.ErrorsBufferSize)
	}
	if w.config.StopOnError && w.errorsBuf == nil {
		w.errorsBuf = make(chan error, w.config.StopOnErrorErrorsBufferSize)
	}
	if w.tasks == nil {
		w.tasks = make(chan Task[R])
	}
}

// initPoolIfNeeded sets up the pool, wiring worker errors to either errorsBuf or errors.
func (w *Workers[R]) initPoolIfNeeded() {
	if w.pool != nil {
		return
	}
	workerErrors := w.errors
	if w.config.StopOnError {
		workerErrors = w.errorsBuf
	}
	newWorkerFn := func() interface{} { return newWorker(w.results, workerErrors) }
	if w.config.MaxWorkers > 0 {
		w.pool = pool.NewFixed(w.config.MaxWorkers, newWorkerFn)
	} else {
		w.pool = pool.NewDynamic(newWorkerFn)
	}
}

// initContext creates the internal context and close channel.
func (w *Workers[R]) initContext(ctx context.Context) {
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.closeCh = make(chan struct{})
}

// startErrorForwarderIfNeeded launches the StopOnError forwarder goroutine if enabled.
func (w *Workers[R]) startErrorForwarderIfNeeded() {
	if !w.config.StopOnError {
		return
	}
	w.forwarderWG.Add(1)
	go func() {
		defer w.forwarderWG.Done()
		forwardedFirst := false
		for {
			select {
			case e := <-w.errorsBuf:
				// Cancel first so dispatch loop stops promptly.
				w.cancel()
				if !forwardedFirst {
					// Forward only the first error outward.
					forwardedFirst = true
					select {
					case w.errors <- e:
						// forwarded synchronously
					default:
						w.errorsSendWG.Add(1)
						go func(err error) {
							defer w.errorsSendWG.Done()
							select {
							case w.errors <- err:
								// delivered when reader appears
							case <-w.closeCh:
								// drop if closing
							}
						}(e)
					}
				}
			case <-w.closeCh:
				// Drain any remaining internal errors (drop them), then exit.
				for {
					select {
					case <-w.errorsBuf:
						// drop
					default:
						return
					}
				}
			}
		}
	}()
}

// startDispatcher launches the task dispatcher loop.
func (w *Workers[R]) startDispatcher(ctx context.Context) {
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				// stop dispatcher without mutating w.tasks to avoid races
				return
			case t := <-w.tasks:
				w.inflight.Add(1)
				go func(tt Task[R]) {
					defer w.inflight.Done()
					w.dispatch(ctx, tt)
				}(t)
			}
		}
	}(ctx)
}

// Close stops scheduling new work, waits for in-flight tasks to finish, then closes results and errors.
//
// Semantics:
// - Idempotent and safe for concurrent use.
// - Cancels the internal context created at Start, causing the dispatcher to stop.
// - Waits for all in-flight task executions to complete.
// - In StopOnError mode, drains any buffered internal errors and forwards them best-effort before closing.
// - Finally closes results and errors channels owned by this instance.
func (w *Workers[R]) Close() {
	w.closeOnce.Do(func() {
		// prevent further AddTask attempts from succeeding via context cancellation (no need to nil tasks)

		// cancel internal context to stop dispatch
		if w.cancel != nil {
			w.cancel()
		}

		// wait for in-flight tasks to finish
		w.inflight.Wait()

		// signal detached senders and forwarder to exit, then wait for them
		if w.closeCh != nil {
			close(w.closeCh)
		}

		// ensure forwarder stopped before closing outward errors
		w.forwarderWG.Wait()

		w.errorsSendWG.Wait()

		// drain any remaining internal errors best-effort
		w.drainInternalErrors()

		close(w.results)
		close(w.errors)
	})
}

// drainInternalErrors forwards any buffered internal errors to the outward channel best-effort.
// Non-blocking send; drops if saturated. Safe to call when errorsBuf is nil.
func (w *Workers[R]) drainInternalErrors() {
	if w.errorsBuf == nil {
		return
	}
	for {
		select {
		case e := <-w.errorsBuf:
			select {
			case w.errors <- e:
				// forwarded
			default:
				// drop
			}
		default:
			return
		}
	}
}

// AddTask adds a task to the Workers queue.
func (w *Workers[R]) AddTask(t Task[R]) error {
	switch {
	case w.tasks == nil:
		return ErrInvalidState

	case cap(w.tasks) > 0 && len(w.tasks) == cap(w.tasks):
		panic("tasks channel is full")
	}

	// If we've been started and the internal context is canceled, don't block; return ErrInvalidState.
	if w.ctx != nil {
		// If already canceled, fail fast deterministically.
		if w.ctx.Err() != nil {
			return ErrInvalidState
		}
		// Otherwise send, but remain cancellation-aware while sending (for unbuffered or saturated channel).
		select {
		case w.tasks <- t:
			return nil
		case <-w.ctx.Done():
			return ErrInvalidState
		}
	}

	// Not started yet (ctx is nil) but tasks channel exists (e.g., constructed with non-zero buffer).
	// Fall back to a normal send.
	w.tasks <- t
	return nil
}

// GetResults returns a channel to receive tasks execution results.
func (w *Workers[R]) GetResults() chan R { return w.results }

// GetErrors returns a channel to receive tasks execution errors.
func (w *Workers[R]) GetErrors() chan error { return w.errors }

func (w *Workers[R]) dispatch(ctx context.Context, t Task[R]) {
	ww := w.pool.Get().(*worker[R])
	ww.execute(ctx, t)
	w.pool.Put(ww)
}
