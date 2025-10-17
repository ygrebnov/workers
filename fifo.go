//go:build ignore

package workers

import (
	"context"
	"sync"
)

// fifoWorkers is a simple FIFO executor that runs tasks sequentially in submission order.
// It implements the Workers interface without using any pool; a single goroutine executes tasks one by one.
// It honors Config.StartImmediately, Config.TasksBufferSize, and Config.StopOnError.
// Results and errors are delivered via the same channel semantics as regular workers.New.
//
// Note: FIFO is intentionally single-threaded to preserve strict ordering.
// This is useful as a baseline for comparisons with pooled executors.

type fifoWorkers[R interface{}] struct {
	config *Config

	once sync.Once

	tasks   chan task[R]
	results chan R
	errors  chan error
}

// NewFIFO creates a new FIFO Workers executor.
// If config is nil, defaults are used. If StartImmediately is true, processing starts right away.
func NewFIFO[R interface{}](ctx context.Context, config *Config) Workers[R] {
	if config == nil {
		config = &Config{}
	}

	// Buffers mirror workers.New defaults.
	r := make(chan R, 1024)

	eCapacity := 1024
	if config.StopOnError {
		eCapacity = 100
	}
	e := make(chan error, eCapacity)

	tasks := make(chan task[R], config.TasksBufferSize)
	if config.TasksBufferSize == 0 {
		tasks = nil // to return error in AddTask until Start.
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

// Start starts sequential processing of queued tasks in FIFO order.
func (w *fifoWorkers[R]) Start(ctx context.Context) {
	w.once.Do(func() {
		var cancel context.CancelFunc
		if w.config.StopOnError {
			ctx, cancel = context.WithCancel(ctx)
		}

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
					// Execute sequentially to preserve FIFO.
					result, err := t.execute(ctx)
					if err != nil {
						w.errors <- err
						if w.config.StopOnError && cancel != nil {
							cancel()
							return
						}
						continue
					}

					if _, ok := t.(*taskError[R]); !ok {
						w.results <- result
					}
				}
			}
		}()
	})
}

// AddTask enqueues a task for sequential execution.
func (w *fifoWorkers[R]) AddTask(t interface{}) error {
	tt, err := newTask[R](t)
	if err != nil {
		return err
	}

	switch {
	case w.tasks == nil:
		// Keep the same error message as the standard workers to simplify diagnosis.
		return ErrInvalidState
	case cap(w.tasks) > 0 && len(w.tasks) == cap(w.tasks):
		panic("tasks channel is full")
	}

	w.tasks <- tt
	return nil
}

// GetResults returns a channel to receive tasks execution results.
func (w *fifoWorkers[R]) GetResults() chan R { return w.results }

// GetErrors returns a channel to receive tasks execution errors.
func (w *fifoWorkers[R]) GetErrors() chan error { return w.errors }
