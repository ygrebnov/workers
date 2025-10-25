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
	cancel   func()
	inflight *sync.WaitGroup
	// dispatcherWG ensures the dispatcher is fully stopped before waiting for inflight tasks
	dispatcherWG  *sync.WaitGroup
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
	dispatcherWG *sync.WaitGroup,
) *lifecycleCoordinator {
	return &lifecycleCoordinator{
		cancel:        cancel,
		inflight:      inflight,
		dispatcherWG:  dispatcherWG,
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
// 2) wait for dispatcher to stop pulling/adding work
// 3) wait inflight tasks to finish
// 4) close closeCh to stop detached senders/forwarders
// 5) wait forwarderWG and errorsSendWG
// 6) drain remaining internal errors best-effort
// 7) close events and wait reorderer
// 8) close results, then errors
func (lc *lifecycleCoordinator) Close() {
	lc.once.Do(func() {
		if lc.cancel != nil {
			lc.cancel()
		}
		// Ensure no further inflight.Add can happen by waiting for dispatcher to exit first
		if lc.dispatcherWG != nil {
			lc.dispatcherWG.Wait()
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
