package workers

import (
	"context"
	"sync"
)

// errorForwarder consumes internal worker errors (in) and, on the first error,
// cancels the context via cancel() and forwards exactly one error to the outward
// errors channel (out). If out is not immediately writable, it uses a detached
// sender goroutine tracked by sendWG that will either deliver later or drop on
// closeCh. After closeCh is closed, it drains any remaining internal errors and exits.
//
// This mirrors the behavior implemented inline in Workers.startErrorForwarderIfNeeded.
// The owner controls lifecycle: errorForwarder does not close any channels.

type errorForwarder struct {
	in      <-chan error    // internal errors (errorsBuf)
	out     chan<- error    // outward errors
	closeCh <-chan struct{} // closed during Close()
	cancel  context.CancelFunc
	sendWG  *sync.WaitGroup // tracks detached sender goroutines
}

func newErrorForwarder(
	in <-chan error, out chan<- error, closeCh <-chan struct{}, cancel context.CancelFunc, sendWG *sync.WaitGroup,
) *errorForwarder {
	return &errorForwarder{in: in, out: out, closeCh: closeCh, cancel: cancel, sendWG: sendWG}
}

func (f *errorForwarder) run() {
	forwardedFirst := false
	for {
		select {
		case e := <-f.in:
			// Cancel first so dispatch loop stops promptly.
			f.cancel()
			if !forwardedFirst {
				forwardedFirst = true
				select {
				case f.out <- e:
					// forwarded synchronously
				default:
					f.sendWG.Add(1)
					go func(err error) {
						defer f.sendWG.Done()
						select {
						case f.out <- err:
							// delivered when reader appears
						case <-f.closeCh:
							// drop if closing
						}
					}(e)
				}
			}
		case <-f.closeCh:
			// Drain any remaining internal errors (drop them), then exit.
			for {
				select {
				case <-f.in:
					// drop
				default:
					return
				}
			}
		}
	}
}
