package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func TestTaskTypeString(t *testing.T) {
	c := workers.New[string](context.Background(), &workers.Config{TasksBufferSize: 5})

	// Valid (string, error)
	require.NoError(t, c.AddTask(workers.TaskFunc[string](func(context.Context) (string, error) { return "", nil })))
	// Valid string-only
	require.NoError(t, c.AddTask(workers.TaskValue[string](func(context.Context) string { return "" })))
	// Valid error-only for string workers (no result emitted)
	require.NoError(t, c.AddTask(workers.TaskError[string](func(context.Context) error { return nil })))
}

type testStruct struct{}

func TestTaskTypePointerStruct(t *testing.T) {
	c := workers.New[*testStruct](context.Background(), &workers.Config{TasksBufferSize: 5})

	// Valid (*testStruct, error)
	require.NoError(t, c.AddTask(workers.TaskFunc[*testStruct](func(context.Context) (*testStruct, error) { return &testStruct{}, nil })))
	// Valid *testStruct only
	require.NoError(t, c.AddTask(workers.TaskValue[*testStruct](func(context.Context) *testStruct { return &testStruct{} })))
	// Valid error-only for *testStruct workers (no result emitted)
	require.NoError(t, c.AddTask(workers.TaskError[*testStruct](func(context.Context) error { return nil })))
}

func TestTaskTypeInterface(t *testing.T) {
	c := workers.New[interface{}](context.Background(), &workers.Config{TasksBufferSize: 10})

	// Valid (interface{}, error)
	require.NoError(t, c.AddTask(workers.TaskFunc[interface{}](func(context.Context) (interface{}, error) { return nil, nil })))
	// Valid interface{} only
	require.NoError(t, c.AddTask(workers.TaskValue[interface{}](func(context.Context) interface{} { return nil })))
	// Valid error-only for interface{} workers (no result emitted)
	require.NoError(t, c.AddTask(workers.TaskError[interface{}](func(context.Context) error { return nil })))
}

func TestTaskTypeFunc(t *testing.T) {
	c := workers.New[func()](context.Background(), &workers.Config{TasksBufferSize: 10})

	require.NoError(t, c.AddTask(workers.TaskFunc[func()](func(context.Context) (func(), error) { return nil, nil })))
	require.NoError(t, c.AddTask(workers.TaskValue[func()](func(context.Context) func() { return nil })))
	require.NoError(t, c.AddTask(workers.TaskError[func()](func(context.Context) error { return nil })))
}
