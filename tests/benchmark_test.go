package tests

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/ygrebnov/workers"
)

func fn(n int) func(context.Context) string {
	return func(context.Context) string {
		ints := make([]int, n)

		for i := range ints {
			ints[i] = i + i
		}

		s := make([]string, n)
		for i := range s {
			s[i] = strconv.Itoa(ints[i])
		}

		return strings.Join(s, "|")
	}
}

func getTasks(start, end, step int) []interface{} {
	tasks := make([]interface{}, 0, (end-start)/step)
	for i := start; i < end; i += step {
		tasks = append(tasks, fn(i))
	}
	return tasks
}

func BenchmarkWorkers(b *testing.B) {
	tests := []struct {
		name             string
		maxWorkers       uint
		bufferSize       int
		startImmediately bool
		tasks            []interface{}
	}{
		// less big tasks, start immediately.
		{
			name:             "fixed_less_big_start_immediately",
			maxWorkers:       uint(runtime.NumCPU()),
			startImmediately: true,
			tasks:            getTasks(10_000_000, 100_000_000, 10_000_000),
		},
		{
			name:             "dynamic_less_big_start_immediately",
			startImmediately: true,
			tasks:            getTasks(10_000_000, 100_000_000, 10_000_000),
		},

		// less big tasks, accumulate.
		{
			name:       "fixed_less_big_accumulate",
			maxWorkers: uint(runtime.NumCPU()),
			bufferSize: 9,
			tasks:      getTasks(10_000_000, 100_000_000, 10_000_000),
		},
		{
			name:       "dynamic_less_big_accumulate",
			bufferSize: 9,
			tasks:      getTasks(10_000_000, 100_000_000, 10_000_000),
		},

		// more small tasks, start immediately.
		{
			name:             "fixed_more_small_start_immediately",
			maxWorkers:       uint(runtime.NumCPU()),
			startImmediately: true,
			tasks:            getTasks(100, 5000, 2),
		},
		{
			name:             "dynamic_more_small_start_immediately",
			startImmediately: true,
			tasks:            getTasks(100, 5000, 2),
		},

		// more small tasks, accumulate tasks.
		{
			name:       "fixed_more_small_accumulate",
			maxWorkers: uint(runtime.NumCPU()),
			bufferSize: 2450,
			tasks:      getTasks(100, 5000, 2),
		},
		{
			name:       "dynamic_more_small_accumulate",
			bufferSize: 2450,
			tasks:      getTasks(100, 5000, 2),
		},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			for range b.N {
				w := workers.New[string](
					context.Background(),
					&workers.Config{
						MaxWorkers:       test.maxWorkers,
						StartImmediately: test.startImmediately,
						TasksBufferSize:  uint(test.bufferSize),
					},
				)

				wg := sync.WaitGroup{}

				go func() {
					for range len(test.tasks) {
						select {
						case <-w.GetResults():
						case <-w.GetErrors():
						}
						wg.Done()
					}
				}()

				for _, task := range test.tasks {
					wg.Add(1)
					err := w.AddTask(task)
					if err != nil {
						b.Fatal(err)
					}
				}

				if !test.startImmediately {
					w.Start(context.Background())
				}

				wg.Wait()

				close(w.GetResults())
				close(w.GetErrors())
			}
		})
	}
}
