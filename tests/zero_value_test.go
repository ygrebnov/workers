package tests

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

func TestZeroValue_AddTaskBeforeStart_ReturnsInvalidState(t *testing.T) {
	var w workers.Workers[int] // zero-value

	err := w.AddTask(workers.TaskValue[int](func(ctx context.Context) int { return 1 }))
	if err == nil || !errors.Is(err, workers.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
}

func TestZeroValue_Start_InitializesDefaults_AndRunsTasks(t *testing.T) {
	ctx := context.Background()
	var w workers.Workers[int] // zero-value

	w.Start(ctx)

	// Enqueue several tasks.
	const n = 5
	for i := 0; i < n; i++ {
		x := i // capture loop variable
		if err := w.AddTask(workers.TaskValue[int](func(ctx context.Context) int { return x * x })); err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
	}

	// Collect exactly n results (with a timeout) before closing to avoid racy inflight accounting.
	results := make([]int, 0, n)
	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()
	for len(results) < n {
		select {
		case r := <-w.GetResults():
			results = append(results, r)
		case err := <-w.GetErrors():
			t.Fatalf("unexpected error: %v", err)
		case <-timer.C:
			t.Fatalf("timed out waiting for results: got %d/%d", len(results), n)
		}
	}

	// Now close and ensure channels are closed.
	w.Close()

	// After Close, both channels must be closed.
	select {
	case _, ok := <-w.GetResults():
		if ok {
			t.Fatalf("results should be closed after Close()")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout waiting for results channel to close")
	}
	select {
	case _, ok := <-w.GetErrors():
		if ok {
			t.Fatalf("errors should be closed after Close()")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout waiting for errors channel to close")
	}

	sort.Ints(results)
	expected := []int{0, 1, 4, 9, 16}
	if !reflect.DeepEqual(expected, results) {
		t.Fatalf("unexpected results: got=%v want=%v", results, expected)
	}
}
