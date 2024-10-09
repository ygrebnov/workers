package workers

import (
	"context"
	"fmt"
)

type worker[R interface{}] struct {
	results chan R
	errors  chan error
}

func newWorker[R interface{}](results chan R, errors chan error) *worker[R] {
	return &worker[R]{results: results, errors: errors}
}

func (w *worker[R]) execute(ctx context.Context, t task[R]) {
	defer func() {
		if ePanic := recover(); ePanic != nil {
			w.errors <- fmt.Errorf("task execution panicked: %v", ePanic)
		}
	}()

	result, err := t.execute(ctx)

	if err != nil {
		w.errors <- err
		return
	}

	if _, ok := t.(*taskError[R]); !ok {
		w.results <- result
	}
}
