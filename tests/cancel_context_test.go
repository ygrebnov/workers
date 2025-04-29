package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ygrebnov/workers"
)

func Test_CancelContext_TaskStringError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel() // Ensure the context is canceled to release resources.

	w := workers.New[string](
		ctx,
		&workers.Config{
			StartImmediately: true,
		},
	)

	done := make(chan struct{}, 1)

	go func() {
		var nErrors int
		results := make([]string, 0, 5)

		for range 5 {
			select {
			case result := <-w.GetResults():
				results = append(results, result)
			case err := <-w.GetErrors():
				nErrors++
				require.ErrorIs(t, err, context.DeadlineExceeded)
			}
		}

		require.Len(t, results, 3, "Expected 3 results")
		require.ElementsMatch(t, results, []string{
			"Executed for: 1, result: 1.",
			"Executed for: 2, result: 4.",
			"Executed for: 3, result: 9.",
		})

		require.Equal(t, 2, nErrors, "Expected 2 errors due to context cancellation")

		done <- struct{}{}
	}()

	for i := 1; i <= 5; i++ {
		err := w.AddTask(newTaskStringError(i, false, false))
		require.NoError(t, err)
	}

	<-done

	done2 := make(chan struct{}, 1)

	go func() {
		timer := time.NewTimer(2 * time.Second)
		for range 5 {
			select {
			case <-timer.C:
				done2 <- struct{}{}
				return
			case result := <-w.GetResults():
				t.Errorf("Unexpected message via results channel: %s", result)
			case err := <-w.GetErrors():
				t.Errorf("Unexpected message via errors channel: %s", err)
			}
		}

		done2 <- struct{}{}
	}()

	for i := 1; i <= 5; i++ {
		err := w.AddTask(newTaskStringError(i, false, false))
		require.ErrorIs(t, err, workers.ErrWorkersStopped)
	}

	<-done2
}

func Test_CancelContext_TaskString(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel() // Ensure the context is canceled to release resources.

	w := workers.New[string](
		ctx,
		&workers.Config{
			StartImmediately: true,
		},
	)

	done := make(chan struct{}, 1)

	go func() {
		var nErrors int
		results := make([]string, 0, 5)

		for range 5 {
			select {
			case result := <-w.GetResults():
				results = append(results, result)
			case err := <-w.GetErrors():
				nErrors++
				require.ErrorIs(t, err, context.DeadlineExceeded)
			}
		}

		require.Len(t, results, 3, "Expected 3 results")
		require.ElementsMatch(t, results, []string{
			"Executed for: 1, result: 1.",
			"Executed for: 2, result: 4.",
			"Executed for: 3, result: 9.",
		})

		require.Equal(t, 2, nErrors, "Expected 2 errors due to context cancellation")

		done <- struct{}{}
	}()

	for i := 1; i <= 5; i++ {
		err := w.AddTask(newTaskString(i, false, false))
		require.NoError(t, err)
	}

	<-done

	done2 := make(chan struct{}, 1)

	go func() {
		timer := time.NewTimer(2 * time.Second)
		for range 5 {
			select {
			case <-timer.C:
				done2 <- struct{}{}
				return
			case result := <-w.GetResults():
				t.Errorf("Unexpected message via results channel: %s", result)
			case err := <-w.GetErrors():
				t.Errorf("Unexpected message via errors channel: %s", err)
			}
		}

		done2 <- struct{}{}
	}()

	for i := 1; i <= 5; i++ {
		err := w.AddTask(newTaskString(i, false, false))
		require.ErrorIs(t, err, workers.ErrWorkersStopped)
	}

	<-done2
}

func Test_CancelContext_TaskError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel() // Ensure the context is canceled to release resources.

	w := workers.New[string](
		ctx,
		&workers.Config{
			StartImmediately: true,
		},
	)

	done := make(chan struct{}, 1)

	go func() {
		var nErrors int
		timer := time.NewTimer(5 * time.Second)

		for range 5 {
			select {
			case <-timer.C:
				if nErrors != 2 {
					t.Errorf("Expected 2 errors due to context cancellation, got: %d", nErrors)
				}
				done <- struct{}{}
				return
			case result := <-w.GetResults():
				t.Errorf("Unexpected message via results channel: %s", result)
			case err := <-w.GetErrors():
				nErrors++
				require.ErrorIs(t, err, context.DeadlineExceeded)
			}
		}

		if nErrors != 2 {
			t.Errorf("Expected 2 errors due to context cancellation, got: %d", nErrors)
		}
		done <- struct{}{}
	}()

	for i := 1; i <= 5; i++ {
		err := w.AddTask(newTaskErr(i, false))
		require.NoError(t, err)
	}

	<-done

	done2 := make(chan struct{}, 1)

	go func() {
		timer := time.NewTimer(2 * time.Second)
		for range 5 {
			select {
			case <-timer.C:
				done2 <- struct{}{}
				return
			case result := <-w.GetResults():
				t.Errorf("Unexpected message via results channel: %s", result)
			case err := <-w.GetErrors():
				t.Errorf("Unexpected message via errors channel: %s", err)
			}
		}

		done2 <- struct{}{}
	}()

	for i := 1; i <= 5; i++ {
		err := w.AddTask(newTaskErr(i, false))
		require.ErrorIs(t, err, workers.ErrWorkersStopped)
	}

	<-done2
}
