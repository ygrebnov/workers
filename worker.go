package workers

import (
	"context"
)

type worker[R any] struct {
	results    chan R
	errors     chan error
	tagEnabled bool
	preserve   bool
	events     chan completionEvent[R]
}

func newWorker[R any](results chan R, errors chan error, tagEnabled bool, preserve bool, events chan completionEvent[R]) *worker[R] {
	return &worker[R]{results: results, errors: errors, tagEnabled: tagEnabled, preserve: preserve, events: events}
}

func (w *worker[R]) execute(ctx context.Context, t Task[R]) {
	result, err := t.Run(ctx)

	if err != nil {
		// Safety-net: ensure error is tagged if not already, only when tagging enabled.
		if w.tagEnabled {
			if _, ok := ExtractTaskIndex(err); !ok {
				if _, okID := ExtractTaskID(err); !okID {
					idx, _ := t.Index()
					err = newTaskTaggedError(err, t.ID(), idx)
				}
			}
		}
		// In preserve-order mode, still emit a completion event (present=false) so reordering can advance.
		if w.preserve && w.events != nil {
			idx, _ := t.Index()
			w.events <- completionEvent[R]{idx: idx, id: t.ID(), present: false}
		}
		w.errors <- err
		return
	}

	if w.preserve && w.events != nil {
		// Emit completion event; present only if task wants to send result.
		idx, _ := t.Index()
		if t.SendResult() {
			w.events <- completionEvent[R]{idx: idx, id: t.ID(), val: result, present: true}
		} else {
			w.events <- completionEvent[R]{idx: idx, id: t.ID(), present: false}
		}
		return
	}

	if t.SendResult() {
		w.results <- result
	}
}
