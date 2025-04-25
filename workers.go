package workers

import (
	"context"
	"sync"

	"github.com/ygrebnov/workers/pool"
)

type Config struct {
	// MaxWorkers defines workers pool maximum size.
	// Zero (default) means that the size will be set dynamically.
	// Zero value is suitable for the majority of cases.
	MaxWorkers uint

	// StartImmediately defines whether workers start executing tasks immediately or not.
	StartImmediately bool

	// StopOnError stops tasks execution if an error occurs.
	StopOnError bool

	TasksBufferSize uint
}

type Workers[R interface{}] interface {
	Start(context.Context)
	AddTask(interface{}) error
	GetResults() chan R
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

func New[R interface{}](ctx context.Context, config *Config) Workers[R] {
	if config == nil {
		config = &Config{}
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

	var w Workers[R]
	if config.StopOnError {
		w = &workersStoppable[R]{
			workers: &workers[R]{
				config:  config,
				tasks:   make(chan task[R], config.TasksBufferSize),
				results: r,
				errors:  make(chan error, 1024),
				pool:    p,
			},
			errorsBuf: e,
		}
	} else {
		w = &workers[R]{
			config:  config,
			tasks:   make(chan task[R], config.TasksBufferSize),
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

func (w *workers[R]) Start(ctx context.Context) {
	w.once.Do(func() {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return

				case t := <-w.tasks:
					go w.dispatch(ctx, t)
				}
			}
		}()
	})
}

func (w *workersStoppable[R]) Start(ctx context.Context) {
	w.once.Do(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)

		go func() {
			for {
				select {
				case <-ctx.Done():
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

func (w *workers[R]) AddTask(t interface{}) error {
	tt, err := newTask[R](t)
	if err != nil {
		return err
	}

	w.tasks <- tt
	return nil
}

func (w *workers[R]) GetResults() chan R {
	return w.results
}

func (w *workers[R]) GetErrors() chan error {
	return w.errors
}

func (w *workers[R]) dispatch(ctx context.Context, t task[R]) {
	ww := w.pool.Get().(*worker[R])
	ww.execute(ctx, t)
	w.pool.Put(ww)
}
