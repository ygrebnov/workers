package workers

import (
	"context"
)

type worker[R any] struct {
	results    chan R
	errors     chan error
	tagEnabled bool
}

func newWorker[R any](results chan R, errors chan error, tagEnabled bool) *worker[R] {
	return &worker[R]{results: results, errors: errors, tagEnabled: tagEnabled}
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
		w.errors <- err
		return
	}

	if t.SendResult() {
		w.results <- result
	}
}
