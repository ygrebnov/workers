package workers

import (
	"context"
	"testing"
)

func TestNew_InvalidOptions_ReturnsError(t *testing.T) {
	t.Parallel()

	// Conflicting pool options should result in an error from New.
	w, err := New[int](
		context.Background(),
		WithFixedPool(1),
		WithDynamicPool(),
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
