package tests

import (
	"context"
	"runtime"
	"strconv"
	"strings"
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
		name       string
		maxWorkers uint
		tasks      []interface{}
	}{
		// less big tasks.
		{"fixed_less_big", uint(runtime.NumCPU()), getTasks(100000, 1000000, 100000)},
		{"dynamic_less_big", 0, getTasks(100000, 1000000, 100000)},
		//{"fixed4_n8_size10_wait50", uint(runtime.NumCPU()), tasksN8Size10},
		//{"dynamic_n8_size10_wait50", 0, tasksN8Size10},

		// more small tasks.
		{"fixed_more_small", uint(runtime.NumCPU()), getTasks(100, 500, 2)},
		{"dynamic_more_small", 0, getTasks(100, 500, 2)},
		//{"fixed4_n256_size2_wait50", uint(runtime.NumCPU()), tasksN256Size2},
		//{"dynamic_n256_size2_wait50", 0, tasksN256Size2},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			for range b.N {
				w := workers.New[string](
					context.Background(),
					&workers.Config{MaxWorkers: test.maxWorkers, StartImmediately: true},
				)

				go func() {
					for range len(test.tasks) {
						select {
						case <-w.GetResults():
						case <-w.GetErrors():
						}
					}
				}()

				for _, task := range test.tasks {
					err := w.AddTask(task)
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
