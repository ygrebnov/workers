package workers

import (
	"github.com/ygrebnov/errorc"
)

// config holds Workers configuration.
type config struct {
	// MaxWorkers defines workers pool maximum size.
	// Zero (default) means that the size will be set dynamically.
	// Zero value is suitable for the majority of cases.
	// Default: 0 (dynamic pool)
	MaxWorkers uint

	// StartImmediately defines whether workers start executing tasks immediately or not.
	// Default: false
	StartImmediately bool

	// StopOnError stops tasks execution if an error occurs.
	// Default: false
	StopOnError bool

	// TasksBufferSize defines the size of the tasks channel buffer.
	// Default: 0 (unbuffered)
	TasksBufferSize uint

	// ResultsBufferSize defines the size of the results channel buffer.
	// Default: 1024.
	ResultsBufferSize uint

	// ErrorsBufferSize defines the size of the outgoing errors channel buffer.
	// Default: 1024.
	ErrorsBufferSize uint

	// StopOnErrorErrorsBufferSize defines the size of the internal errors buffer used
	// when StopOnError is enabled. Smaller buffer triggers cancellation quickly.
	// Default: 100.
	StopOnErrorErrorsBufferSize uint

	// ErrorTagging enables wrapping task errors with task metadata (ID and index).
	// When enabled, any error returned by a task is wrapped to support correlation.
	// Default: false (disabled).
	ErrorTagging bool

	// PreserveOrder enforces emitting results in the same order as tasks were added.
	// When enabled, Workers reorder completed tasks and only deliver results to the outward
	// results channel in input order (indices assigned at AddTask). This may reduce throughput
	// due to head-of-line blocking and increases memory for buffering.
	// Default: false (disabled).
	PreserveOrder bool
}

// defaultConfig centralizes default values for config.
// These defaults are applied by both initialize (when cfg is nil) and New (options builder base).
func defaultConfig() config {
	return config{
		MaxWorkers:                  0,     // dynamic pool
		StartImmediately:            false, // explicit Start by default
		StopOnError:                 false,
		TasksBufferSize:             0,
		ResultsBufferSize:           1024,
		ErrorsBufferSize:            1024,
		StopOnErrorErrorsBufferSize: 100,
		ErrorTagging:                false,
		PreserveOrder:               false,
	}
}

// validateConfig performs lightweight invariants checks.
// It returns nil for all currently valid states; reserved for future validation expansions.
func validateConfig(_ *config) error {
	// MaxWorkers == 0 -> dynamic pool; >0 -> fixed-size pool.
	// All buffer sizes are uint; zero is a valid choice (unbuffered) except we provide non-zero defaults above.
	// No hard validation required at the moment.
	return nil
}

// Option configures Workers. Use New(ctx, opts...) to construct Workers via options.
// Breaking change: Option now returns an error on invalid input instead of panicking.
type Option func(*config) error

// // Internal builder state for options assembly.
// type configOptions struct {
// 	cfg          config
// 	poolSelected poolType
// }
//
// type poolType int
//
// const (
// 	poolUnspecified poolType = iota
// 	poolDynamic
// 	poolFixed
// )
//
// var errFixedDynamicPoolOptionsConflict = errorc.With(
// 	ErrInvalidOption,
// 	errorc.String(
// 		"",
// 		"conflicting pool options: WithFixedPool and WithDynamicPool both specified",
// 	),
// )

// WithFixedPool selects a fixed-size worker pool with the given capacity (must be > 0).
func WithFixedPool(n uint) Option {
	return func(cfg *config) error {
		if n == 0 {
			return errorc.With(ErrInvalidConfig, errorc.String("", "WithFixedPool requires n > 0"))
		}
		cfg.MaxWorkers = n
		return nil
	}
}

// WithTasksBuffer sets the size of the tasks channel buffer.
func WithTasksBuffer(size uint) Option {
	return func(cfg *config) error { cfg.TasksBufferSize = size; return nil }
}

// WithResultsBuffer sets the size of the results channel buffer (default 1024).
func WithResultsBuffer(size uint) Option {
	return func(cfg *config) error { cfg.ResultsBufferSize = size; return nil }
}

// WithErrorsBuffer sets the size of the outgoing errors channel buffer (default 1024).
func WithErrorsBuffer(size uint) Option {
	return func(cfg *config) error { cfg.ErrorsBufferSize = size; return nil }
}

// WithStopOnErrorBuffer sets the size of the internal errors buffer used when StopOnError is enabled (default 100).
func WithStopOnErrorBuffer(size uint) Option {
	return func(cfg *config) error { cfg.StopOnErrorErrorsBufferSize = size; return nil }
}

// WithStartImmediately starts workers execution immediately.
func WithStartImmediately() Option {
	return func(cfg *config) error { cfg.StartImmediately = true; return nil }
}

// WithStopOnError stops tasks execution when the first error occurs.
func WithStopOnError() Option {
	return func(cfg *config) error { cfg.StopOnError = true; return nil }
}

// WithErrorTagging enables wrapping task errors with task metadata (ID and index).
func WithErrorTagging() Option {
	return func(cfg *config) error { cfg.ErrorTagging = true; return nil }
}

// WithPreserveOrder enforces emitting results in input order at the core level.
// When enabled, Workers reorder completed tasks and only deliver results in index order.
func WithPreserveOrder() Option {
	return func(cfg *config) error { cfg.PreserveOrder = true; return nil }
}
