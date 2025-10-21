package tests

import (
	"context"
	"sync"

	"github.com/ygrebnov/workers"
)

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

	tasks   chan workers.Task[R]
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

	tasks := make(chan workers.Task[R], config.TasksBufferSize)
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
			w.tasks = make(chan workers.Task[R])
		}

		go func() {
			for {
				select {
				case <-ctx.Done():
					w.tasks = nil
					return
				case t := <-w.tasks:
					res, err := t.Run(ctx)
					if err != nil {
						w.errors <- err
						if w.config.StopOnError && cancel != nil {
							cancel()
							return
						}
						continue
					}
					if t.SendResult() {
						w.results <- res
					}
				}
			}
		}()
	})
}

func (w *fifoWorkers[R]) AddTask(t workers.Task[R]) error {
	switch {
	case w.tasks == nil:
		return workers.ErrInvalidState
	case cap(w.tasks) > 0 && len(w.tasks) == cap(w.tasks):
		panic("tasks channel is full")
	}
	w.tasks <- t
	return nil
}

func (w *fifoWorkers[R]) GetResults() chan R    { return w.results }
func (w *fifoWorkers[R]) GetErrors() chan error { return w.errors }
