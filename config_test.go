package workers

import (
	"context"
	"testing"
)

func TestValidateConfig_Defaults(t *testing.T) {
	cfg := defaultConfig()
	if err := validateConfig(&cfg); err != nil {
		t.Fatalf("validateConfig returned error for defaults: %v", err)
	}
}

func TestDefaultConfig_Values(t *testing.T) {
	cfg := defaultConfig()
	if cfg.MaxWorkers != 0 {
		t.Fatalf("MaxWorkers default = %d; want 0", cfg.MaxWorkers)
	}
	if cfg.StartImmediately != false {
		t.Fatalf("StartImmediately default = %v; want false", cfg.StartImmediately)
	}
	if cfg.StopOnError != false {
		t.Fatalf("StopOnError default = %v; want false", cfg.StopOnError)
	}
	if cfg.TasksBufferSize != 0 {
		t.Fatalf("TasksBufferSize default = %d; want 0", cfg.TasksBufferSize)
	}
	if cfg.ResultsBufferSize != 1024 {
		t.Fatalf("ResultsBufferSize default = %d; want 1024", cfg.ResultsBufferSize)
	}
	if cfg.ErrorsBufferSize != 1024 {
		t.Fatalf("ErrorsBufferSize default = %d; want 1024", cfg.ErrorsBufferSize)
	}
	if cfg.StopOnErrorErrorsBufferSize != 100 {
		t.Fatalf("StopOnErrorErrorsBufferSize default = %d; want 100", cfg.StopOnErrorErrorsBufferSize)
	}
}

func TestNew_InvalidOptions_ReturnsError(t *testing.T) {
	t.Parallel()

	// Conflicting pool options should result in an error from New.
	w, err := New[int](
		context.Background(),
		WithFixedPool(0),
	)
	if err == nil {
		t.Fatalf("expected error from New with conflicting options, got nil (w=%v)", w)
	}
	if w != nil {
		t.Fatalf("expected nil workers on error, got: %v", w)
	}
}

func TestNew_ValidOptions_Succeeds(t *testing.T) {
	t.Parallel()

	w, err := New[int](
		context.Background(),
		WithFixedPool(1),
		WithStartImmediately(),
		WithTasksBuffer(4),
		WithResultsBuffer(8),
		WithErrorsBuffer(8),
	)
	if err != nil {
		t.Fatalf("unexpected error from NewOptions with valid options: %v", err)
	}
	if w == nil {
		t.Fatalf("expected non-nil workers instance")
	}
}
