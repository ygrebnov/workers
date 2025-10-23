package workers

import (
	"context"
	"sync"

	"github.com/ygrebnov/workers/pool"
)

// dispatcher reads tasks from the input channel and executes them via executor.
// It tracks inflight tasks with a WaitGroup. The dispatcher stops when ctx.Done()
// is closed. It never closes channels it doesn't own and doesn't drain tasks
// after cancellation (mirrors current Workers.startDispatcher semantics).

type dispatcher[R any] struct {
	tasks    <-chan Task[R]
	inflight *sync.WaitGroup
	pool     pool.Pool // worker pool
}

func newDispatcher[R any](tasks <-chan Task[R], inflight *sync.WaitGroup, p pool.Pool) *dispatcher[R] {
	return &dispatcher[R]{tasks: tasks, inflight: inflight, pool: p}
}

// run starts the dispatch loop and returns when the context is canceled.
func (d *dispatcher[R]) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// stop dispatcher without mutating tasks channel or draining
			return
		case t := <-d.tasks:
			// account in-flight and execute in a goroutine
			d.inflight.Add(1)
			go func(tt Task[R]) {
				defer d.inflight.Done()
				d.execute(ctx, tt)
			}(t)
		}
	}
}

func (d *dispatcher[R]) execute(ctx context.Context, t Task[R]) {
	ww := d.pool.Get().(*worker[R])
	ww.execute(ctx, t)
	d.pool.Put(ww)
}
