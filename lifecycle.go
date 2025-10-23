package workers

import (
	"sync"
)

// lifecycleCoordinator encapsulates the shutdown sequence for Workers.
// It is a wiring helper: it doesn't own channels; it orchestrates cancellation,
// waits, draining, and channel closures in a deterministic order.
//
// Close() is safe for concurrent calls; the sequence executes exactly once.

type lifecycleCoordinator struct {
	cancel        func()
	inflight      *sync.WaitGroup
	closeCh       chan struct{}
	forwarderWG   *sync.WaitGroup
	errorsSendWG  *sync.WaitGroup
	drainInternal func()
	closeEvents   func()
	waitReorderer func()
	closeResults  func()
	closeErrors   func()

	once sync.Once
}

func newLifecycleCoordinator(
	cancel func(),
	inflight *sync.WaitGroup,
	closeCh chan struct{},
	forwarderWG *sync.WaitGroup,
	errorsSendWG *sync.WaitGroup,
	drainInternal func(),
	closeEvents func(),
	waitReorderer func(),
	closeResults func(),
	closeErrors func(),
) *lifecycleCoordinator {
	return &lifecycleCoordinator{
		cancel:        cancel,
		inflight:      inflight,
		closeCh:       closeCh,
		forwarderWG:   forwarderWG,
		errorsSendWG:  errorsSendWG,
		drainInternal: drainInternal,
		closeEvents:   closeEvents,
		waitReorderer: waitReorderer,
		closeResults:  closeResults,
		closeErrors:   closeErrors,
	}
}

// Close executes the shutdown sequence exactly once:
// 1) cancel internal context
// 2) wait inflight tasks
// 3) close closeCh to stop detached senders/forwarders
// 4) wait forwarderWG and errorsSendWG
// 5) drain remaining internal errors best-effort
// 6) close events and wait reorderer
// 7) close results, then errors
func (lc *lifecycleCoordinator) Close() {
	lc.once.Do(func() {
		if lc.cancel != nil {
			lc.cancel()
		}
		if lc.inflight != nil {
			lc.inflight.Wait()
		}
		if lc.closeCh != nil {
			close(lc.closeCh)
		}
		if lc.forwarderWG != nil {
			lc.forwarderWG.Wait()
		}
		if lc.errorsSendWG != nil {
			lc.errorsSendWG.Wait()
		}
		if lc.drainInternal != nil {
			lc.drainInternal()
		}
		if lc.closeEvents != nil {
			lc.closeEvents()
		}
		if lc.waitReorderer != nil {
			lc.waitReorderer()
		}
		if lc.closeResults != nil {
			lc.closeResults()
		}
		if lc.closeErrors != nil {
			lc.closeErrors()
		}
	})
}
