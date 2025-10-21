package tests

import (
	"context"
	"runtime"
	"sync"

	"github.com/ygrebnov/workers"
)

type concurrentWorkers[R any] struct {
	config *workers.Config

	once sync.Once

	tasks   chan workers.Task[R]
	results chan R
	errors  chan error
}

// newMT creates a new private multithreaded executor that does not preserve order.
// Concurrency is config.MaxWorkers; if zero, defaults to runtime.NumCPU().
func newMT[R any](ctx context.Context, config *workers.Config) testWorkers[R] {
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

	w := &concurrentWorkers[R]{
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

func (w *concurrentWorkers[R]) Start(ctx context.Context) {
	w.once.Do(func() {
		var cancel context.CancelFunc
		if w.config.StopOnError {
			ctx, cancel = context.WithCancel(ctx)
		}

		if w.tasks == nil {
			w.tasks = make(chan workers.Task[R])
		}

		workersN := int(w.config.MaxWorkers)
		if workersN <= 0 {
			workersN = runtime.NumCPU()
		}

		wg := sync.WaitGroup{}
		wg.Add(workersN)

		for i := 0; i < workersN; i++ {
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
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
		}

		// Optional: allow a separate goroutine to wait on wg and do nothing; channels remain open for benchmark harness to close.
		go func() {
			wg.Wait()
		}()
	})
}

func (w *concurrentWorkers[R]) AddTask(t workers.Task[R]) error {
	switch {
	case w.tasks == nil:
		return workers.ErrInvalidState
	case cap(w.tasks) > 0 && len(w.tasks) == cap(w.tasks):
		panic("tasks channel is full")
	}
	w.tasks <- t
	return nil
}

func (w *concurrentWorkers[R]) GetResults() chan R    { return w.results }
func (w *concurrentWorkers[R]) GetErrors() chan error { return w.errors }
