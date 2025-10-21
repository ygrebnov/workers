package workers

import (
	"context"

	"github.com/ygrebnov/errorc"
)

// Option configures Workers. Use NewOptions(ctx, opts...) to construct Workers via options.
// Breaking change: Option now returns an error on invalid input instead of panicking.
type Option func(*configOptions) error

// Internal builder state for options assembly.
type configOptions struct {
	cfg          Config
	poolSelected poolType
}

type poolType int

const (
	poolUnspecified poolType = iota
	poolDynamic
	poolFixed
)

var errFixedDynamicPoolOptionsConflict = errorc.With(
	ErrInvalidWorkersOption,
	errorc.String(
		"",
		"conflicting pool options: WithFixedPool and WithDynamicPool both specified",
	),
)

// WithFixedPool selects a fixed-size worker pool with the given capacity (must be > 0).
func WithFixedPool(n uint) Option {
	return func(co *configOptions) error {
		if co.poolSelected != poolUnspecified && co.poolSelected != poolFixed {
			return errFixedDynamicPoolOptionsConflict
		}
		if n == 0 {
			return errorc.With(ErrInvalidWorkersOption, errorc.String("", "WithFixedPool requires n > 0"))
		}
		co.poolSelected = poolFixed
		co.cfg.MaxWorkers = n
		return nil
	}
}

// WithDynamicPool selects a dynamic-size worker pool (the default if no pool option is provided).
func WithDynamicPool() Option {
	return func(co *configOptions) error {
		if co.poolSelected != poolUnspecified && co.poolSelected != poolDynamic {
			return errFixedDynamicPoolOptionsConflict
		}
		co.poolSelected = poolDynamic
		co.cfg.MaxWorkers = 0
		return nil
	}
}

// WithTasksBuffer sets the size of the tasks channel buffer.
func WithTasksBuffer(size uint) Option {
	return func(co *configOptions) error { co.cfg.TasksBufferSize = size; return nil }
}

// WithResultsBuffer sets the size of the results channel buffer (default 1024).
func WithResultsBuffer(size uint) Option {
	return func(co *configOptions) error { co.cfg.ResultsBufferSize = size; return nil }
}

// WithErrorsBuffer sets the size of the outgoing errors channel buffer (default 1024).
func WithErrorsBuffer(size uint) Option {
	return func(co *configOptions) error { co.cfg.ErrorsBufferSize = size; return nil }
}

// WithStopOnErrorBuffer sets the size of the internal errors buffer used when StopOnError is enabled (default 100).
func WithStopOnErrorBuffer(size uint) Option {
	return func(co *configOptions) error { co.cfg.StopOnErrorErrorsBufferSize = size; return nil }
}

// WithStartImmediately starts workers execution immediately.
func WithStartImmediately() Option {
	return func(co *configOptions) error { co.cfg.StartImmediately = true; return nil }
}

// WithStopOnError stops tasks execution when the first error occurs.
func WithStopOnError() Option {
	return func(co *configOptions) error { co.cfg.StopOnError = true; return nil }
}

// NewOptions creates a new Workers instance using functional options.
// Breaking change: returns (*Workers, error) instead of panicking on invalid options.
// It preserves backward compatibility in behavior by internally constructing a Config and delegating to New.
func NewOptions[R interface{}](ctx context.Context, opts ...Option) (*Workers[R], error) {
	co := configOptions{cfg: defaultConfig(), poolSelected: poolUnspecified}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&co); err != nil {
			return nil, errorc.With(ErrInvalidWorkersOption, errorc.Error("", err))
		}
	}

	// If pool type not specified, default to dynamic (same as MaxWorkers == 0 today).
	if co.poolSelected == poolUnspecified {
		co.poolSelected = poolDynamic
		co.cfg.MaxWorkers = 0
	}

	if err := validateConfig(&co.cfg); err != nil {
		return nil, errorc.With(ErrInvalidWorkersConfig, errorc.Error("", err))
	}

	return New[R](ctx, &co.cfg), nil
}
