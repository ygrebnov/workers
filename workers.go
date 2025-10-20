package workers

import (
	"context"
	"errors"
	"sync"

	"github.com/ygrebnov/workers/pool"
)

// Config holds Workers configuration.
type Config struct {
	// MaxWorkers defines workers pool maximum size.
	// Zero (default) means that the size will be set dynamically.
	// Zero value is suitable for the majority of cases.
	// Default: 0 (dynamic pool)
	MaxWorkers uint

	// StartImmediately defines whether workers start executing tasks immediately or not.
	// Default: false
	StartImmediately bool

	// StopOnError stops tasks execution if an error occurs.
	// Default: false
	StopOnError bool

	// TasksBufferSize defines the size of the tasks channel buffer.
	// Default: 0 (unbuffered)
	TasksBufferSize uint

	// ResultsBufferSize defines the size of the results channel buffer.
	// Default: 1024.
	ResultsBufferSize uint

	// ErrorsBufferSize defines the size of the outgoing errors channel buffer.
	// Default: 1024.
	ErrorsBufferSize uint

	// StopOnErrorErrorsBufferSize defines the size of the internal errors buffer used
	// when StopOnError is enabled. Smaller buffer triggers cancellation quickly.
	// Default: 100.
	StopOnErrorErrorsBufferSize uint
}

// Workers is an interface that defines methods on Workers.
type Workers[R interface{}] interface {
	// Start starts the Workers and begins executing tasks.
	// Start may be called only once.
	// In case 'StopOnError' is set to true, tasks execution is stopped on error.
	Start(context.Context)

	// AddTask adds a task to the Workers queue.
	// The task must be a function with one of the following signatures:
	//
	// * func(context.Context) (R, error),
	//
	// * func(context.Context) R,
	//
	// * func(context.Context) error.
	//
	// In case the Workers have been started, the task will be dispatched immediately and
	// executed as soon as a worker is available.
	AddTask(interface{}) error

	// GetResults returns a channel to receive tasks execution results.
	GetResults() chan R

	// GetErrors returns a channel to receive tasks execution errors.
	GetErrors() chan error
}

type workers[R interface{}] struct {
	config *Config

	once sync.Once

	pool pool.Pool

	tasks   chan task[R]
	results chan R
	errors  chan error // outward errors channel

	// When StopOnError is enabled, workers produce into this smaller internal buffer,
	// which Start() drains and forwards into the outward errors channel, then cancels.
	errorsBuf chan error
}

// New creates a new Workers object instance and returns it.
//
// Deprecated: This Config-based constructor will be deprecated in a future release.
// Prefer NewOptions(ctx, opts...) which will become the primary New in the next major version.
//
// The Workers object is not started automatically.
// To start it, either 'StartImmediately' configuration option must be set to true or
// the Start method must be called explicitly.
func New[R interface{}](ctx context.Context, config *Config) Workers[R] {
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

	tasks := make(chan task[R], config.TasksBufferSize)
	if config.TasksBufferSize == 0 {
		tasks = nil // to return error in AddTask.
	}

	w := &workers[R]{
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
func (w *workers[R]) Start(ctx context.Context) {
	w.once.Do(func() {
		if w.tasks == nil {
			w.tasks = make(chan task[R])
		}

		// If StopOnError is enabled, create a cancellable context and forward
		// internal errors to the outward channel. Cancel first to stop scheduling
		// new work; then forward the triggering error. If the outward channel is
		// full, forward in a detached goroutine to avoid blocking cancellation.
		if w.config.StopOnError {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)

			go func(ctx context.Context) {
				for {
					select {
					case <-ctx.Done():
						return
					case e := <-w.errorsBuf:
						// Cancel first so dispatch loop stops promptly.
						cancel()
						// Best-effort, non-blocking forward; if full, forward asynchronously.
						select {
						case w.errors <- e:
							// forwarded
						default:
							go func(err error) { w.errors <- err }(e)
						}
					}
				}
			}(ctx)
		}

		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					w.tasks = nil
					return

				case t := <-w.tasks:
					go w.dispatch(ctx, t)
				}
			}
		}(ctx)
	})
}

var (
	ErrInvalidState = errors.New("cannot add a task for non-started workers with unbuffered tasks channel")
)

// AddTask adds a task to the Workers queue.
func (w *workers[R]) AddTask(t interface{}) error {
	tt, err := newTask[R](t)
	if err != nil {
		return err
	}

	switch {
	case w.tasks == nil:
		return ErrInvalidState

	case cap(w.tasks) > 0 && len(w.tasks) == cap(w.tasks):
		panic("tasks channel is full")
	}

	w.tasks <- tt
	return nil
}

// GetResults returns a channel to receive tasks execution results.
func (w *workers[R]) GetResults() chan R {
	return w.results
}

// GetErrors returns a channel to receive tasks execution errors.
func (w *workers[R]) GetErrors() chan error {
	return w.errors
}

func (w *workers[R]) dispatch(ctx context.Context, t task[R]) {
	ww := w.pool.Get().(*worker[R])
	ww.execute(ctx, t)
	w.pool.Put(ww)
}
