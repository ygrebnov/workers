package workers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestTaskAdapters_BasicExecution(t *testing.T) {
	type testCase struct {
		name      string
		mk        func() Task[int]
		expectR   int
		expectErr error
	}

	okErr := error(nil)

	tests := []testCase{
		{
			name:      "TaskFunc -> success",
			mk:        func() Task[int] { return TaskFunc[int](func(_ context.Context) (int, error) { return 7, nil }) },
			expectR:   7,
			expectErr: okErr,
		},
		{
			name:      "TaskValue -> success",
			mk:        func() Task[int] { return TaskValue[int](func(_ context.Context) int { return 5 }) },
			expectR:   5,
			expectErr: okErr,
		},
		{
			name:      "TaskError -> success (nil) returns zero R and nil",
			mk:        func() Task[int] { return TaskError[int](func(_ context.Context) error { return nil }) },
			expectR:   0,
			expectErr: okErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			got, execErr := tt.mk().Run(ctx)
			if execErr != tt.expectErr {
				t.Fatalf("Run error = %v, want %v", execErr, tt.expectErr)
			}
			if got != tt.expectR {
				t.Fatalf("Run result = %v, want %v", got, tt.expectR)
			}
		})
	}
}

func TestTaskFunc_Run_AllBranches(t *testing.T) {
	type testCase struct {
		name      string
		fn        func(context.Context) (int, error)
		expectR   int
		expectErr func(error) bool
	}

	blocker := make(chan struct{})
	defer close(blocker) // safety: ensure no leak if test fails early

	tests := []testCase{
		{
			name:      "success result + nil error",
			fn:        func(_ context.Context) (int, error) { return 10, nil },
			expectR:   10,
			expectErr: func(err error) bool { return err == nil },
		},
		{
			name:      "returned error",
			fn:        func(_ context.Context) (int, error) { return 0, errors.New("boom") },
			expectR:   0,
			expectErr: func(err error) bool { return err != nil && strings.Contains(err.Error(), "boom") },
		},
		{
			name:      "panic is recovered",
			fn:        func(_ context.Context) (int, error) { panic("kaboom") },
			expectR:   0,
			expectErr: func(err error) bool { return err != nil && strings.Contains(err.Error(), "panicked") },
		},
		{
			name: "context cancellation wins",
			fn: func(ctx context.Context) (int, error) {
				select {
				case <-ctx.Done():
					<-blocker
					return 0, nil
				}
			},
			expectR:   0,
			expectErr: func(err error) bool { return err != nil && errors.Is(err, context.Canceled) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			var cancel context.CancelFunc
			if strings.Contains(tt.name, "cancellation") {
				ctx, cancel = context.WithCancel(context.Background())
				cancel() // cancel immediately so execTask selects ctx.Done()
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			}
			defer cancel()

			got, execErr := TaskFunc[int](tt.fn).Run(ctx)
			if got != tt.expectR {
				t.Fatalf("result = %v, want %v", got, tt.expectR)
			}
			if !tt.expectErr(execErr) {
				t.Fatalf("unexpected error: %v", execErr)
			}
		})
	}
}

func TestTaskValue_Run_AllBranches(t *testing.T) {
	type testCase struct {
		name      string
		fn        func(context.Context) int
		expectR   int
		expectErr func(error) bool
	}

	blocker := make(chan struct{})
	defer close(blocker)

	tests := []testCase{
		{
			name:      "success result",
			fn:        func(_ context.Context) int { return 21 },
			expectR:   21,
			expectErr: func(err error) bool { return err == nil },
		},
		{
			name:      "panic is recovered",
			fn:        func(_ context.Context) int { panic("oops") },
			expectR:   0,
			expectErr: func(err error) bool { return err != nil && strings.Contains(err.Error(), "panicked") },
		},
		{
			name: "context cancellation wins",
			fn: func(ctx context.Context) int {
				select {
				case <-ctx.Done():
					<-blocker
					return 0
				}
			},
			expectR:   0,
			expectErr: func(err error) bool { return err != nil && errors.Is(err, context.Canceled) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			var cancel context.CancelFunc
			if strings.Contains(tt.name, "cancellation") {
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			}
			defer cancel()

			got, execErr := TaskValue[int](tt.fn).Run(ctx)
			if got != tt.expectR {
				t.Fatalf("result = %v, want %v", got, tt.expectR)
			}
			if !tt.expectErr(execErr) {
				t.Fatalf("unexpected error: %v", execErr)
			}
		})
	}
}

func TestTaskError_Run_AllBranches(t *testing.T) {
	type testCase struct {
		name      string
		fn        func(context.Context) error
		expectR   int
		expectErr func(error) bool
	}

	blocker := make(chan struct{})
	defer close(blocker)

	tests := []testCase{
		{
			name:      "success nil error",
			fn:        func(_ context.Context) error { return nil },
			expectR:   0,
			expectErr: func(err error) bool { return err == nil },
		},
		{
			name:      "returned error",
			fn:        func(_ context.Context) error { return errors.New("sad") },
			expectR:   0,
			expectErr: func(err error) bool { return err != nil && strings.Contains(err.Error(), "sad") },
		},
		{
			name:      "panic is recovered",
			fn:        func(_ context.Context) error { panic("boom") },
			expectR:   0,
			expectErr: func(err error) bool { return err != nil && strings.Contains(err.Error(), "panicked") },
		},
		{
			name: "context cancellation wins",
			fn: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					<-blocker
					return nil
				}
			},
			expectR:   0,
			expectErr: func(err error) bool { return err != nil && errors.Is(err, context.Canceled) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			var cancel context.CancelFunc
			if strings.Contains(tt.name, "cancellation") {
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			}
			defer cancel()

			got, execErr := TaskError[int](tt.fn).Run(ctx)
			if got != tt.expectR {
				t.Fatalf("result = %v, want %v", got, tt.expectR)
			}
			if !tt.expectErr(execErr) {
				t.Fatalf("unexpected error: %v", execErr)
			}
		})
	}
}
