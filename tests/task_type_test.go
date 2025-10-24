package tests

import (
	"context"
	"testing"

	"github.com/ygrebnov/workers"
)

func TestTaskTypeString(t *testing.T) {
	c := workers.New[string](context.Background(), &workers.Config{TasksBufferSize: 5})

	// Valid (string, error)
	if err := c.AddTask(workers.TaskFunc[string](func(context.Context) (string, error) { return "", nil })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	// Valid string-only
	if err := c.AddTask(workers.TaskValue[string](func(context.Context) string { return "" })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	// Valid error-only for string workers (no result emitted)
	if err := c.AddTask(workers.TaskError[string](func(context.Context) error { return nil })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
}

type testStruct struct{}

func TestTaskTypePointerStruct(t *testing.T) {
	c := workers.New[*testStruct](context.Background(), &workers.Config{TasksBufferSize: 5})

	// Valid (*testStruct, error)
	if err := c.AddTask(workers.TaskFunc[*testStruct](func(context.Context) (*testStruct, error) { return &testStruct{}, nil })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	// Valid *testStruct only
	if err := c.AddTask(workers.TaskValue[*testStruct](func(context.Context) *testStruct { return &testStruct{} })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	// Valid error-only for *testStruct workers (no result emitted)
	if err := c.AddTask(workers.TaskError[*testStruct](func(context.Context) error { return nil })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
}

func TestTaskTypeInterface(t *testing.T) {
	c := workers.New[interface{}](context.Background(), &workers.Config{TasksBufferSize: 10})

	// Valid (interface{}, error)
	if err := c.AddTask(workers.TaskFunc[interface{}](func(context.Context) (interface{}, error) { return nil, nil })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	// Valid interface{} only
	if err := c.AddTask(workers.TaskValue[interface{}](func(context.Context) interface{} { return nil })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	// Valid error-only for interface{} workers (no result emitted)
	if err := c.AddTask(workers.TaskError[interface{}](func(context.Context) error { return nil })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
}

func TestTaskTypeFunc(t *testing.T) {
	c := workers.New[func()](context.Background(), &workers.Config{TasksBufferSize: 10})

	if err := c.AddTask(workers.TaskFunc[func()](func(context.Context) (func(), error) { return nil, nil })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := c.AddTask(workers.TaskValue[func()](func(context.Context) func() { return nil })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := c.AddTask(workers.TaskError[func()](func(context.Context) error { return nil })); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
}
