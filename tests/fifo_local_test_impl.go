package tests

import (
	"context"
	"sync"

	"github.com/ygrebnov/workers"
)

type runnable[R any] struct {
	fn         func(context.Context) (R, error)
	sendResult bool
}

func makeRunnable[R any](fn interface{}) (runnable[R], error) {
	switch typed := fn.(type) {
	case func(context.Context) (R, error):
		return runnable[R]{fn: typed, sendResult: true}, nil
	case func(context.Context) R:
		return runnable[R]{fn: func(ctx context.Context) (R, error) { return typed(ctx), nil }, sendResult: true}, nil
	case func(context.Context) error:
		return runnable[R]{fn: func(ctx context.Context) (R, error) { var zero R; return zero, typed(ctx) }, sendResult: false}, nil
	default:
		return runnable[R]{}, workers.ErrInvalidTaskType
	}
}

// fifoWorkers is a simple FIFO executor that runs tasks sequentially in submission order.
// It implements the Workers interface without using any pool; a single goroutine executes tasks one by one.
// It honors Config.StartImmediately, Config.TasksBufferSize, and Config.StopOnError.
// Results and errors are delivered via the same channel semantics as regular workers.New.
//
// Note: FIFO is intentionally single-threaded to preserve strict ordering.
// This is useful as a baseline for comparisons with pooled executors.
type fifoWorkers[R any] struct {
	config *workers.Config

	once sync.Once

	tasks   chan runnable[R]
	results chan R
	errors  chan error
}

// newFIFO creates a new private FIFO executor used only in tests/benchmarks.
func newFIFO[R any](ctx context.Context, config *workers.Config) workers.Workers[R] {
	if config == nil {
		config = &workers.Config{}
	}

	r := make(chan R, 1024)
	eCap := 1024
	if config.StopOnError {
		eCap = 100
	}
	e := make(chan error, eCap)

	tasks := make(chan runnable[R], config.TasksBufferSize)
	if config.TasksBufferSize == 0 {
		tasks = nil // to return error in AddTask until Start
	}

	w := &fifoWorkers[R]{
		config:  config,
		tasks:   tasks,
		results: r,
		errors:  e,
	}

	if config.StartImmediately {
		w.Start(ctx)
	}

	return w
}

func (w *fifoWorkers[R]) Start(ctx context.Context) {
	w.once.Do(func() {
		var cancel context.CancelFunc
		if w.config.StopOnError {
			ctx, cancel = context.WithCancel(ctx)
		}

		if w.tasks == nil {
			w.tasks = make(chan runnable[R])
		}

		go func() {
			for {
				select {
				case <-ctx.Done():
					w.tasks = nil
					return
				case t := <-w.tasks:
					res, err := t.fn(ctx)
					if err != nil {
						w.errors <- err
						if w.config.StopOnError && cancel != nil {
							cancel()
							return
						}
						continue
					}
					if t.sendResult {
						w.results <- res
					}
				}
			}
		}()
	})
}

func (w *fifoWorkers[R]) AddTask(t interface{}) error {
	r, err := makeRunnable[R](t)
	if err != nil {
		return err
	}
	switch {
	case w.tasks == nil:
		return workers.ErrInvalidState
	case cap(w.tasks) > 0 && len(w.tasks) == cap(w.tasks):
		panic("tasks channel is full")
	}
	w.tasks <- r
	return nil
}

func (w *fifoWorkers[R]) GetResults() chan R    { return w.results }
func (w *fifoWorkers[R]) GetErrors() chan error { return w.errors }
