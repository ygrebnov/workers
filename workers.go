package workers

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ygrebnov/workers/pool"
)

// Workers manages a pool of workers executing typed tasks and exposing results/errors channels.
// Workers is a concrete struct; methods are safe for concurrent use.
// Zero-value is usable: call Start(ctx) to initialize with defaults (or construct via New).
type Workers[R interface{}] struct {
	// noCopy prevents accidental copying of the controller.
	//go:nocopy
	nc noCopy

	config *config

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

	// sequence counter for tasks accepted via AddTask (used for error tagging and preserve-order indexing)
	seq uint64

	// preserve-order internal events stream and coordinator waitgroup
	events    chan completionEvent[R]
	reorderWG sync.WaitGroup
}

// noCopy is a vet-recognized marker to discourage copying types with this field embedded.
// It works with the "-copylocks" analyzer via the presence of Lock/Unlock methods.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// initialize sets up the Workers controller using the provided configuration.
// If cfg is nil, defaults are applied. If StartImmediately is set, Start(ctx) is called.
func (w *Workers[R]) initialize(ctx context.Context, cfg *config) {
	if cfg == nil {
		c := defaultConfig()
		cfg = &c
	}

	r := make(chan R, cfg.ResultsBufferSize)

	// Prepare the channel that workers will write errors to.
	// In StopOnError mode, workers produce into a smaller internal buffer (errorsBuf)
	// which the controller drains and forwards to the outward errors channel.
	var workerErrors chan error
	if cfg.StopOnError {
		workerErrors = make(chan error, cfg.StopOnErrorErrorsBufferSize)
	} else {
		workerErrors = make(chan error, cfg.ErrorsBufferSize)
	}

	// Prepare preserve-order events channel if enabled.
	var events chan completionEvent[R]
	if cfg.PreserveOrder {
		events = make(chan completionEvent[R], cfg.ResultsBufferSize)
	}

	newWorkerFn := func() interface{} {
		return newWorker(r, workerErrors, cfg.ErrorTagging, cfg.PreserveOrder, events)
	}

	var p pool.Pool
	if cfg.MaxWorkers > 0 {
		p = pool.NewFixed(cfg.MaxWorkers, newWorkerFn)
	} else {
		p = pool.NewDynamic(newWorkerFn)
	}

	tasks := make(chan Task[R], cfg.TasksBufferSize)
	if cfg.TasksBufferSize == 0 {
		tasks = nil // to return error in AddTask.
	}

	w.config = cfg
	w.tasks = tasks
	w.results = r
	w.pool = p
	w.events = events

	if cfg.StopOnError {
		// outward errors channel keeps a larger buffer for receivers
		w.errors = make(chan error, cfg.ErrorsBufferSize)
		w.errorsBuf = workerErrors
	} else {
		// in non-stoppable mode, workers write directly to the outward errors channel
		w.errors = workerErrors
	}

	if cfg.StartImmediately {
		w.Start(ctx)
	}
}

// Start starts the Workers and begins executing tasks.
func (w *Workers[R]) Start(ctx context.Context) {
	w.once.Do(func() {
		w.initDefaultsIfNeeded()
		w.initChannelsIfNeeded()
		w.initPoolIfNeeded()
		w.initContext(ctx)
		w.startErrorForwarderIfNeeded()
		w.startReordererIfNeeded()
		d := newDispatcher[R](w.tasks, &w.inflight, w.pool)
		go d.run(w.ctx)
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
	if w.config.PreserveOrder && w.events == nil {
		w.events = make(chan completionEvent[R], w.config.ResultsBufferSize)
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
	newWorkerFn := func() interface{} {
		return newWorker(w.results, workerErrors, w.config.ErrorTagging, w.config.PreserveOrder, w.events)
	}
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
	ef := newErrorForwarder(w.errorsBuf, w.errors, w.closeCh, w.cancel, &w.errorsSendWG)
	go func() {
		defer w.forwarderWG.Done()
		ef.run()
	}()
}

// startReordererIfNeeded launches the preserve-order reorder coordinator when enabled.
func (w *Workers[R]) startReordererIfNeeded() {
	if !w.config.PreserveOrder {
		return
	}
	w.reorderWG.Add(1)
	r := newReorderer[R](w.events, w.results)
	go func() {
		defer w.reorderWG.Done()
		// reorderer ignores context today; pass internal ctx for future-proofing
		r.run(w.ctx)
	}()
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
		lc := newLifecycleCoordinator(
			w.cancel,
			&w.inflight,
			w.closeCh,
			&w.forwarderWG,
			&w.errorsSendWG,
			w.drainInternalErrors,
			func() {
				if w.events != nil {
					close(w.events)
				}
			},
			func() { w.reorderWG.Wait() },
			func() { close(w.results) },
			func() { close(w.errors) },
		)
		lc.Close()
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

	// Assign input index when either error tagging or preserve-order are enabled.
	if w.config != nil && (w.config.ErrorTagging || w.config.PreserveOrder) {
		idx := int(atomic.AddUint64(&w.seq, 1) - 1)
		t = t.WithIndex(idx)
		if w.config.ErrorTagging {
			// wrap to propagate tagging on the error path
			t = wrapTaskWithTagging(t, idx)
		}
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

// wrapTaskWithTagging returns a Task that executes the original and wraps any error
// with task ID and input index metadata for correlation.
func wrapTaskWithTagging[R interface{}](t Task[R], index int) Task[R] {
	origID := t.ID()
	wrap := TaskFunc[R](func(ctx context.Context) (R, error) {
		res, err := t.Run(ctx)
		if err != nil {
			return res, newTaskTaggedError(err, origID, index)
		}
		return res, nil
	})
	// Preserve SendResult and ID semantics
	if !t.SendResult() {
		wrap = TaskError[R](func(ctx context.Context) error {
			_, err := t.Run(ctx)
			if err != nil {
				return newTaskTaggedError(err, origID, index)
			}
			return nil
		})
	}
	return wrap.WithID(origID)
}
