package workers

import "context"

// ForEach applies fn to each item concurrently using the Workers engine.
// It constructs error-only tasks (no result emission) and delegates execution to RunAll,
// returning the aggregated error (errors.Join) or nil when all succeed.
// Options like WithStopOnError, WithPreserveOrder, pool selection, and buffers are honored.
func ForEach[T any](ctx context.Context, items []T, fn func(context.Context, T) error, opts ...Option) error {
	if len(items) == 0 {
		return nil
	}
	// Use a dummy result type for RunAll; tasks created via TaskError do not emit results.
	tasks := make([]Task[struct{}], 0, len(items))
	for i := range items {
		item := items[i] // capture
		tasks = append(tasks, TaskError[struct{}](func(c context.Context) error { return fn(c, item) }))
	}
	_, err := RunAll[struct{}](ctx, tasks, opts...)
	return err
}
