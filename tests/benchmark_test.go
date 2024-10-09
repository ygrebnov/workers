package tests

import (
	"context"
	"testing"

	"github.com/ygrebnov/workers"
)

func BenchmarkWorkers(b *testing.B) {
	tests := []struct {
		name       string
		maxWorkers uint
		tasks      []interface{}
	}{
		// less big tasks
		{"fixed4_n8_size10_wait50", 4, tasksN8Size10},
		{"dynamic_n8_size10_wait50", 0, tasksN8Size10},

		// more small tasks
		{"fixed4_n256_size2_wait50", 4, tasksN256Size2},
		{"dynamic_n256_size2_wait50", 0, tasksN256Size2},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			for range b.N {
				w := workers.New[string](
					context.Background(),
					&workers.Config{MaxWorkers: test.maxWorkers, StartImmediately: true},
				)
				actual := make([]string, 0, len(test.tasks))
				errors := make([]error, 0, len(test.tasks))

				go func() {
					for range len(test.tasks) {
						select {
						case result := <-w.GetResults():
							actual = append(actual, result)
						case err := <-w.GetErrors():
							errors = append(errors, err)
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
