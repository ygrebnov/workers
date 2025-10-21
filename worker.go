package workers

import (
	"context"
)

type worker[R any] struct {
	results chan R
	errors  chan error
}

func newWorker[R any](results chan R, errors chan error) *worker[R] {
	return &worker[R]{results: results, errors: errors}
}

func (w *worker[R]) execute(ctx context.Context, t Task[R]) {
	result, err := t.Run(ctx)

	if err != nil {
		w.errors <- err
		return
	}

	if t.SendResult() {
		w.results <- result
	}
}
