package tests

import (
	"context"
	"fmt"
	"time"
)

func newTaskStringError(n int, e, p bool) func(ctx context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		// Simulate a long-running task
		time.Sleep(time.Millisecond * 600 * time.Duration(n))

		if e && n == 3 {
			// Simulate an error.
			return "", fmt.Errorf("error executing task for: %d", n)
		}
		if p && n == 3 {
			// Simulate a panic.
			panic(fmt.Sprintf("panic on executing task for: %d", n))
		}

		return fmt.Sprintf("Executed for: %d, result: %d.", n, n*n), nil
	}
}

func newTaskString(n int, returnError bool) func(ctx context.Context) string {
	return func(ctx context.Context) string {
		// Simulate a long-running task
		time.Sleep(time.Millisecond * 600 * time.Duration(n))

		// Function returns for n == 3
		if returnError && n == 3 {
			return ""
		}

		return fmt.Sprintf("Executed for: %d, result: %d.", n, n*n)
	}
}

func newTaskErr(n int, returnError bool) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// Simulate a long-running task
		time.Sleep(time.Millisecond * 600 * time.Duration(n))

		// Function returns an error for n == 3
		if returnError && n == 3 {
			return fmt.Errorf("error executing task for: %d", n)
		}

		return nil
	}
}
