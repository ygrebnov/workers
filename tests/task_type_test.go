package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func TestTaskTypeString(t *testing.T) {
	c := workers.New[string](context.Background(), &workers.Config{TasksBufferSize: 5})

	err := c.AddTask(func(ctx context.Context) (string, error) {
		return "test", nil
	})
	require.NoError(t, err, "Failed to add (string, error) task to workers")

	err = c.AddTask(func(ctx context.Context) string {
		return "test"
	})
	require.NoError(t, err, "Failed to add (string) task to workers")

	err = c.AddTask(func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err, "Failed to add (error) task to workers")

	err = c.AddTask(func(ctx context.Context) int {
		return 42
	})
	require.ErrorIs(
		t,
		err,
		workers.ErrInvalidTaskType,
		"Expected invalid task type error when adding (int) task to workers",
	)

	err = c.AddTask(func(ctx context.Context) *string {
		return nil
	})
	require.ErrorIs(
		t,
		err,
		workers.ErrInvalidTaskType,
		"Expected invalid task type error when adding (*string) task to workers",
	)
}

type testStruct struct{}

func TestTaskTypePointerStruct(t *testing.T) {
	c := workers.New[*testStruct](context.Background(), &workers.Config{TasksBufferSize: 5})

	err := c.AddTask(func(ctx context.Context) (*testStruct, error) {
		return &testStruct{}, nil
	})
	require.NoError(t, err, "Failed to add (*testStruct, error) task to workers")

	err = c.AddTask(func(ctx context.Context) *testStruct {
		return &testStruct{}
	})
	require.NoError(t, err, "Failed to add (*testStruct) task to workers")

	err = c.AddTask(func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err, "Failed to add (error) task to workers")

	err = c.AddTask(func(ctx context.Context) testStruct {
		return testStruct{}
	})
	require.ErrorIs(
		t,
		err,
		workers.ErrInvalidTaskType,
		"Expected invalid task type error when adding (testStruct) task to workers",
	)

	err = c.AddTask(func(ctx context.Context) {})
	require.ErrorIs(
		t,
		err,
		workers.ErrInvalidTaskType,
		"Expected invalid task type error when adding (void) task to workers",
	)

	err = c.AddTask(func(ctx context.Context) interface{} {
		return nil
	})
	require.ErrorIs(
		t,
		err,
		workers.ErrInvalidTaskType,
		"Expected invalid task type error when adding (interface{}) task to workers",
	)
}

func TestTaskTypeInterface(t *testing.T) {
	c := workers.New[interface{}](context.Background(), &workers.Config{TasksBufferSize: 10})

	err := c.AddTask(func(ctx context.Context) (interface{}, error) {
		return nil, nil
	})
	require.NoError(t, err, "Failed to add (interface{}, error) task to workers")

	err = c.AddTask(func(ctx context.Context) interface{} {
		return nil
	})
	require.NoError(t, err, "Failed to add (interface) task to workers")

	err = c.AddTask(func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err, "Failed to add (error) task to workers")

	err = c.AddTask(func(ctx context.Context) (string, error) {
		return "test", nil
	})
	require.ErrorIs(
		t,
		err,
		workers.ErrInvalidTaskType,
		"Expected invalid task type error when adding (string, error) task to workers",
	)

	err = c.AddTask(func(ctx context.Context) testStruct {
		return testStruct{}
	})
	require.ErrorIs(
		t,
		err,
		workers.ErrInvalidTaskType,
		"Expected invalid task type error when adding (testStruct) task to workers",
	)

	err = c.AddTask(func(ctx context.Context) {})
	require.ErrorIs(
		t,
		err,
		workers.ErrInvalidTaskType,
		"Expected invalid task type error when adding (void) task to workers",
	)
}

func TestTaskTypeFunc(t *testing.T) {
	c := workers.New[func()](context.Background(), &workers.Config{TasksBufferSize: 10})

	err := c.AddTask(func(ctx context.Context) (func(), error) {
		return nil, nil
	})
	require.NoError(t, err, "Failed to add (func(), error) task to workers")

	err = c.AddTask(func(ctx context.Context) func() {
		return nil
	})
	require.NoError(t, err, "Failed to add (func()) task to workers")

	err = c.AddTask(func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err, "Failed to add (error) task to workers")

	err = c.AddTask(func(ctx context.Context) interface{} {
		return nil
	})
	require.ErrorIs(
		t,
		err,
		workers.ErrInvalidTaskType,
		"Expected invalid task type error when adding (interface{}) task to workers",
	)

	err = c.AddTask(func(ctx context.Context) {})
	require.ErrorIs(
		t,
		err,
		workers.ErrInvalidTaskType,
		"Expected invalid task type error when adding (void) task to workers",
	)
}
