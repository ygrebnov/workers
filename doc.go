// Package workers provides a lightweight way to execute multiple tasks concurrently
// using either a dynamic or a fixed-size worker pool.
//
// Constructors
//   - New(ctx, *Config): current stable constructor that accepts a Config.
//     This form is planned for deprecation in a future release.
//   - NewOptions(ctx, opts ...Option): options-based constructor. This will
//     become the primary New in the next major version. Prefer this in new code.
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
// Channel lifecycle
// The library exposes two channels:
//   - Results: deliver task results (for non-error-only tasks)
//   - Errors: deliver task execution errors
//
// The library does not close these channels automatically. The recommended pattern
// is to drain channels while tasks are running and close them in application code
// once you know there will be no more messages (e.g., after a WaitGroup is done).
//
// Pools
//   - Dynamic pool (default): grows and shrinks as needed via sync.Pool.
//   - Fixed pool: caps the number of concurrently executing workers.
package workers
