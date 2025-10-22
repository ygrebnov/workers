package workers

import "context"

// Map fans out items through fn with the Workers execution engine and returns results and aggregated error.
// Semantics:
// - Delegates to RunAll after wrapping each item into a Task that calls fn(ctx, item).
// - Honors options like WithPreserveOrder, WithStopOnError, WithErrorTagging, pool selection, and buffers.
// - Results ordering follows RunAll + opts: completion order by default; input order if WithPreserveOrder is enabled.
// - When StopOnError is enabled, cancellation is triggered on the first error; some tasks may not start.
func Map[T, R any](
	ctx context.Context,
	items []T,
	fn func(context.Context, T) (R, error),
	opts ...Option,
) ([]R, error) {
	if len(items) == 0 {
		return nil, nil
	}
	// Adapt inputs into tasks.
	tasks := make([]Task[R], 0, len(items))
	for i := range items {
		item := items[i] // capture
		tasks = append(tasks, TaskFunc[R](func(c context.Context) (R, error) { return fn(c, item) }))
	}
	return RunAll[R](ctx, tasks, opts...)
}
