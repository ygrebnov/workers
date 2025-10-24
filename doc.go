// Package workers provides a lightweight way to execute multiple tasks concurrently
// using either a dynamic or a fixed-size worker pool.
//
// Constructors
//   - New(ctx, opts ...Option): options-based constructor (primary).
//     Prefer this form in new code.
//
// Zero value and lifecycle
//   - The zero value of Workers is usable after calling Start(ctx); it initializes with defaults.
//     For custom configuration, construct via New with options.
//   - Start(ctx) is safe to call at most once; concurrent calls are idempotent.
//   - Close() is provided to stop scheduling, wait for in-flight work to finish, and close
//     the results and errors channels. Close is idempotent and safe for concurrent use.
//
// Defaults
// Unless overridden, the following defaults apply to a newly created instance:
//   - MaxWorkers: 0 (dynamic pool)
//   - StartImmediately: false (explicit Start is required if TasksBufferSize == 0)
//   - StopOnError: false
//   - TasksBufferSize: 0
//   - ResultsBufferSize: 1024
//   - ErrorsBufferSize: 1024
//   - StopOnErrorErrorsBufferSize: 100
//
// Channels and Close semantics
// The library exposes two channels owned by a Workers instance:
//   - Results: delivers task results (for non-error-only tasks)
//   - Errors: delivers task execution errors
//
// Closing strategy:
//   - If you call Close(), the library will:
//   - cancel the internal context so no new tasks are dispatched,
//   - wait for in-flight tasks to finish,
//   - forward any buffered internal errors (when StopOnError is enabled), then
//   - close both Results and Errors channels.
//     Do not manually close the channels if you use Close(); double-closing panics in Go.
//   - If you manage lifecycle manually (advanced use), you can close channels yourself once
//     you're certain no further sends can occur. In that case, do not call Close().
//
// StopOnError behavior
//   - When StopOnError is enabled, the first error cancels the controller promptly to stop scheduling.
//     The triggering error is forwarded to the outward Errors channel. If that channel is full, the
//     error is dropped (best-effort forwarding) to avoid blocking cancellation and avoid goroutine leaks.
//     Close() forwards any remaining buffered internal errors best-effort before closing the outward
//     Errors channel; saturated outward buffers may still drop some errors.
//
// Pools
//   - Dynamic pool (default): grows and shrinks as needed via sync.Pool.
//   - Fixed pool: caps the number of concurrently executing workers.
package workers
