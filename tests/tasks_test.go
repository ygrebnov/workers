package tests

import (
	"context"
	"fmt"
	"time"
)

func newTaskStringError(n int, long, e, p bool) func(ctx context.Context) (string, error) {
	return func(context.Context) (string, error) {
		if long {
			// Simulate a long-running task
			time.Sleep(time.Millisecond * 60 * time.Duration(n))
		}

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

func newTaskString(n int, long, e, p bool) func(ctx context.Context) string {
	return func(context.Context) string {
		if long {
			// Simulate a long-running task
			time.Sleep(time.Millisecond * 60 * time.Duration(n))
		}

		if e && n == 3 {
			// Simulate an error.
			return ""
		}
		if p && n == 3 {
			// Simulate a panic.
			panic(fmt.Sprintf("panic on executing task for: %d", n))
		}

		return fmt.Sprintf("Executed for: %d, result: %d.", n, n*n)
	}
}

func newTaskErr(n int, long, e, p bool) func(ctx context.Context) error {
	return func(context.Context) error {
		if long {
			// Simulate a long-running task
			time.Sleep(time.Millisecond * 60 * time.Duration(n))
		}

		if e && n == 3 {
			// Simulate an error.
			return fmt.Errorf("error executing task for: %d", n)
		}
		if p && n == 3 {
			// Simulate a panic.
			panic(fmt.Sprintf("panic on executing task for: %d", n))
		}

		return nil
	}
}
