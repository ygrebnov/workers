package workers

import (
	"context"
	"fmt"
)

// Option configures Workers. Use NewWithOptions(ctx, opts...) to construct Workers via options.
type Option func(*configOptions)

// internal builder state for options assembly.
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

// defaultConfig centralizes default values for Config.
// This acts as the library's "model" of defaults.
func defaultConfig() Config {
	return Config{
		MaxWorkers:                  0,
		StartImmediately:            false,
		StopOnError:                 false,
		TasksBufferSize:             0,
		ResultsBufferSize:           1024,
		ErrorsBufferSize:            1024,
		StopOnErrorErrorsBufferSize: 100,
	}
}

// WithFixedPool selects a fixed-size worker pool with the given capacity (must be > 0).
func WithFixedPool(n uint) Option {
	return func(co *configOptions) {
		if co.poolSelected != poolUnspecified && co.poolSelected != poolFixed {
			panic("conflicting pool options: WithFixedPool and WithDynamicPool both specified")
		}
		if n == 0 {
			panic("WithFixedPool requires n > 0")
		}
		co.poolSelected = poolFixed
		co.cfg.MaxWorkers = n
	}
}

// WithDynamicPool selects a dynamic-size worker pool (the default if no pool option is provided).
func WithDynamicPool() Option {
	return func(co *configOptions) {
		if co.poolSelected != poolUnspecified && co.poolSelected != poolDynamic {
			panic("conflicting pool options: WithFixedPool and WithDynamicPool both specified")
		}
		co.poolSelected = poolDynamic
		co.cfg.MaxWorkers = 0
	}
}

// WithTasksBuffer sets the size of the tasks channel buffer.
func WithTasksBuffer(size uint) Option {
	return func(co *configOptions) { co.cfg.TasksBufferSize = size }
}

// WithResultsBuffer sets the size of the results channel buffer (default 1024).
func WithResultsBuffer(size uint) Option {
	return func(co *configOptions) { co.cfg.ResultsBufferSize = size }
}

// WithErrorsBuffer sets the size of the outgoing errors channel buffer (default 1024).
func WithErrorsBuffer(size uint) Option {
	return func(co *configOptions) { co.cfg.ErrorsBufferSize = size }
}

// WithStopOnErrorBuffer sets the size of the internal errors buffer used when StopOnError is enabled (default 100).
func WithStopOnErrorBuffer(size uint) Option {
	return func(co *configOptions) { co.cfg.StopOnErrorErrorsBufferSize = size }
}

// WithStartImmediately starts workers execution immediately.
func WithStartImmediately() Option { return func(co *configOptions) { co.cfg.StartImmediately = true } }

// WithStopOnError stops tasks execution when the first error occurs.
func WithStopOnError() Option { return func(co *configOptions) { co.cfg.StopOnError = true } }

// NewWithOptions creates a new Workers instance using functional options.
// It preserves backward compatibility by internally constructing a Config and delegating to New.
func NewWithOptions[R interface{}](ctx context.Context, opts ...Option) Workers[R] {
	co := configOptions{cfg: defaultConfig(), poolSelected: poolUnspecified}
	for _, opt := range opts {
		if opt == nil {
			panic("nil workers option")
		}
		opt(&co)
	}

	// If pool type not specified, default to dynamic (same as MaxWorkers == 0 today).
	if co.poolSelected == poolUnspecified {
		co.poolSelected = poolDynamic
		co.cfg.MaxWorkers = 0
	}

	// Final sanity checks mirroring existing Config semantics.
	if co.poolSelected == poolFixed && co.cfg.MaxWorkers == 0 {
		panic(fmt.Errorf("invalid fixed pool size: %d", co.cfg.MaxWorkers))
	}

	return New[R](ctx, &co.cfg)
}
