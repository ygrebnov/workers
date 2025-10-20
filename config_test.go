package workers

import "testing"

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
