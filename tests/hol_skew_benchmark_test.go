package tests

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/ygrebnov/workers"
)

// Benchmark to visualize head-of-line blocking under WithPreserveOrder on skewed durations.
// It reports a custom metric first10_ms to indicate time-to-first-10 results.
func BenchmarkHOL_Skewed_Durations(b *testing.B) {
	mkTasks := func(n, slowAt int, slow, fast time.Duration) []workers.Task[int] {
		ts := make([]workers.Task[int], 0, n)
		for i := 0; i < n; i++ {
			d := fast
			if i < slowAt { // place a few slow tasks at the beginning
				d = slow
			}
			ts = append(ts, workers.TaskValue[int](func(ctx context.Context) int {
				t := time.NewTimer(d)
				select {
				case <-t.C:
				case <-ctx.Done():
				}
				return i
			}))
		}
		return ts
	}

	bench := func(b *testing.B, opts ...workers.Option) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			tasks := mkTasks(64, 1, 50*time.Millisecond, 1*time.Millisecond)
			// Always start immediately; pool selection defaults to dynamic unless provided via opts.
			base := []workers.Option{workers.WithStartImmediately()}
			w, err := workers.NewOptions[int](ctx, append(base, opts...)...)
			if err != nil {
				b.Fatalf("NewOptions failed: %v", err)
			}

			firstK := 10
			seen := 0
			start := time.Now()
			firstKDur := time.Duration(0)

			// reader
			doneCh := make(chan struct{})
			go func() {
				defer close(doneCh)
				for seen < len(tasks) {
					select {
					case <-w.GetResults():
						seen++
						if seen == firstK && firstKDur == 0 {
							firstKDur = time.Since(start)
						}
					case <-w.GetErrors():
						seen++
					}
				}
			}()

			for _, t := range tasks {
				if err := w.AddTask(t); err != nil {
					b.Fatalf("AddTask failed: %v", err)
				}
			}

			<-doneCh
			// close channels to avoid goroutine leaks in the harness
			close(w.GetResults())
			close(w.GetErrors())

			b.ReportMetric(float64(firstKDur.Milliseconds()), "first10_ms")
		}
	}

	b.Run("dynamic", func(b *testing.B) { bench(b) })
	b.Run("dynamic_preserve_order", func(b *testing.B) { bench(b, workers.WithPreserveOrder()) })
	b.Run("fixed_NCPU", func(b *testing.B) {
		bench(b, workers.WithFixedPool(uint(runtime.NumCPU())))
	})
	b.Run("fixed_NCPU_preserve_order", func(b *testing.B) {
		bench(b, workers.WithFixedPool(uint(runtime.NumCPU())), workers.WithPreserveOrder())
	})
}
