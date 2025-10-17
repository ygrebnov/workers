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
	MaxWorkers uint

	// StartImmediately defines whether workers start executing tasks immediately or not.
	StartImmediately bool

	// StopOnError stops tasks execution if an error occurs.
	StopOnError bool

	// TasksBufferSize defines the size of the tasks channel buffer.
	TasksBufferSize uint
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
	errors  chan error
}

type workersStoppable[R interface{}] struct {
	*workers[R]

	errorsBuf chan error
}

// New creates a new Workers object instance and returns it.
// The Workers object is not started automatically.
// To start it, either 'StartImmediately' configuration option must be set to true or
// the Start method must be called explicitly.
func New[R interface{}](ctx context.Context, config *Config) Workers[R] {
	if config == nil {
		cfg := defaultConfig()
		config = &cfg
	}

	r := make(chan R, 1024)

	eCapacity := 1024
	if config.StopOnError {
		eCapacity = 100
	}
	e := make(chan error, eCapacity)

	newWorkerFn := func() interface{} {
		return newWorker(r, e)
	}

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

	var w Workers[R]
	if config.StopOnError {
		w = &workersStoppable[R]{
			workers: &workers[R]{
				config:  config,
				tasks:   tasks,
				results: r,
				errors:  make(chan error, 1024),
				pool:    p,
			},
			errorsBuf: e,
		}
	} else {
		w = &workers[R]{
			config:  config,
			tasks:   tasks,
			results: r,
			errors:  e,
			pool:    p,
		}
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
		go func() {
			for {
				select {
				case <-ctx.Done():
					w.tasks = nil
					return

				case t := <-w.tasks:
					go w.dispatch(ctx, t)
				}
			}
		}()
	})
}

// Start starts the Workers and begins executing tasks.
// In case 'StopOnError' is set to true, tasks execution is stopped on error.
func (w *workersStoppable[R]) Start(ctx context.Context) {
	w.once.Do(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)

		if w.tasks == nil {
			w.tasks = make(chan task[R])
		}

		go func() {
			for {
				select {
				case <-ctx.Done():
					w.tasks = nil
					return

				case t := <-w.tasks:
					go w.dispatch(ctx, t)

				case e := <-w.errorsBuf:
					w.errors <- e

					if w.config.StopOnError {
						cancel()
					}
				}
			}
		}()
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
