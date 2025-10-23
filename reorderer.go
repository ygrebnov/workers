package workers

import "context"

// reorderer is a small internal component that enforces result emission order.
// It consumes completion events and emits results to the provided sink strictly
// in the original input order. See preserve_order.go for the detailed contract.
//
// The reorderer runs in a single goroutine via run() and never closes the results
// channel; shutdown is coordinated by the owner by closing the events channel.
//
// Generic over result type R.
// Not exported; tested via package-level tests.

type reorderer[R any] struct {
	// inputs/outputs
	events  <-chan completionEvent[R]
	results chan<- R
}

func newReorderer[R any](events <-chan completionEvent[R], results chan<- R) *reorderer[R] {
	return &reorderer[R]{events: events, results: results}
}

// run executes the coordinator loop until the events channel is closed.
// It maintains an in-order cursor and small in-memory buffers for out-of-order
// completions and no-result markers.
func (r *reorderer[R]) run(_ context.Context) {
	// Local state mirrors the previous inline implementation.
	next := 0
	buf := make(map[int]R)
	seenNoRes := make(map[int]struct{})

	for ev := range r.events {
		if ev.present {
			buf[ev.idx] = ev.val
		} else {
			seenNoRes[ev.idx] = struct{}{}
		}
		// Flush contiguous from the current cursor.
		next = r.flushContiguous(next, buf, seenNoRes)
	}

	// Best-effort final flush of contiguous tail after events closed.
	r.finalFlush(&next, buf, seenNoRes)
}

// flushContiguous emits consecutive results/no-result markers starting from `next`.
// Returns the advanced cursor value.
func (r *reorderer[R]) flushContiguous(next int, buf map[int]R, seenNoRes map[int]struct{}) int {
	for {
		if v, ok := buf[next]; ok {
			// emit result and advance
			r.results <- v
			delete(buf, next)
			next++
			continue
		}
		if _, ok := seenNoRes[next]; ok {
			// advance without emission
			delete(seenNoRes, next)
			next++
			continue
		}
		break
	}
	return next
}

// finalFlush performs a best-effort contiguous flush after events are closed.
// This mirrors the original behavior: only a contiguous prefix from `next` can
// be emitted; gaps stop the flush.
func (r *reorderer[R]) finalFlush(next *int, buf map[int]R, seenNoRes map[int]struct{}) {
	*next = r.flushContiguous(*next, buf, seenNoRes)
}
