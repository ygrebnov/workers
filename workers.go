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
		// If this instance was not constructed via New/NewOptions, lazily initialize defaults.
		if w.config == nil {
			cfg := defaultConfig()
			w.config = &cfg
		}

		// Initialize results and errors channels if missing.
		if w.results == nil {
			w.results = make(chan R, w.config.ResultsBufferSize)
		}

		// For StopOnError, workers write to an internal buffer that is forwarded outward.
		// Otherwise, workers write directly to the outward errors channel.
		if w.config.StopOnError {
			if w.errorsBuf == nil {
				w.errorsBuf = make(chan error, w.config.StopOnErrorErrorsBufferSize)
			}
			if w.errors == nil {
				w.errors = make(chan error, w.config.ErrorsBufferSize)
			}
		} else {
			if w.errors == nil {
				w.errors = make(chan error, w.config.ErrorsBufferSize)
			}
		}

		// Initialize pool if missing.
		if w.pool == nil {
			// Workers should write errors to errorsBuf if StopOnError, else directly to outward errors.
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

		// create internal context that Close() can cancel
		w.ctx, w.cancel = context.WithCancel(ctx)

		// initialize closeCh used to stop detached senders during Close
		w.closeCh = make(chan struct{})

		if w.tasks == nil {
			w.tasks = make(chan Task[R])
		}

		// If StopOnError is enabled, forward internal errors to the outward channel.
		if w.config.StopOnError {
			w.forwarderWG.Add(1)
			go func(ctx context.Context) {
				defer w.forwarderWG.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case e := <-w.errorsBuf:
						// Cancel first so dispatch loop stops promptly.
						w.cancel()
						// Try immediate forward; if it would block, forward asynchronously with Close-aware cancellation.
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
				}
			}(w.ctx)
		}

		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					w.tasks = nil
					return

				case t := <-w.tasks:
					w.inflight.Add(1)
					go func(tt Task[R]) {
						defer w.inflight.Done()
						w.dispatch(ctx, tt)
					}(t)
				}
			}
		}(w.ctx)
	})
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
		// prevent further AddTask attempts from succeeding
		w.tasks = nil

		// cancel internal context to stop dispatch and forwarder
		if w.cancel != nil {
			w.cancel()
		}

		// wait for in-flight tasks to finish
		w.inflight.Wait()

		// ensure forwarder stopped before closing outward errors
		w.forwarderWG.Wait()

		// signal detached senders to exit and wait for them
		if w.closeCh != nil {
			close(w.closeCh)
		}
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
