package tests

import (
	"context"
	"runtime"
	"sync"
	"testing"

	"github.com/ygrebnov/workers"
)

// cpuHeavyTask returns a task that multiplies two n x n matrices of float64 and returns the sum of the result elements.
func cpuHeavyTask(n int) func(context.Context) float64 {
	return func(ctx context.Context) float64 {
		// Allocate and initialize matrices A and B.
		A := make([]float64, n*n)
		B := make([]float64, n*n)
		C := make([]float64, n*n)

		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				A[i*n+j] = float64(i+j+1) * 0.5
				B[i*n+j] = float64(i-j+1) * 0.25
			}
		}

		// Standard triple-loop matrix multiplication.
		for i := 0; i < n; i++ {
			for k := 0; k < n; k++ {
				a := A[i*n+k]
				for j := 0; j < n; j++ {
					C[i*n+j] += a * B[k*n+j]
				}
			}
		}

		// Sum the result to produce a deterministic output and prevent dead-code elimination.
		var sum float64
		for i := 0; i < len(C); i++ {
			sum += C[i]
		}
		return sum
	}
}

// memHeavyTask returns a task that allocates and fills a buffer of size bytes and returns the checksum.
func memHeavyTask(size int) func(context.Context) float64 {
	return func(ctx context.Context) float64 {
		buf := make([]byte, size)
		var x byte = 1
		for i := range buf {
			x = x*33 + byte(i)
			buf[i] = x
		}
		var sum uint64
		for _, b := range buf {
			sum += uint64(b)
		}
		return float64(sum)
	}
}

func runBench[R any](b *testing.B, tasks []workers.Task[R], mk func() testWorkers[R]) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := mk()

		wg := sync.WaitGroup{}
		wg.Add(len(tasks))

		// Reader goroutine that drains results and errors until all tasks complete.
		go func() {
			for range len(tasks) {
				select {
				case <-w.GetResults():
					wg.Done()
				case <-w.GetErrors():
					wg.Done()
				}
			}
		}()

		for _, t := range tasks {
			if err := w.AddTask(t); err != nil {
				b.Fatalf("AddTask failed: %v", err)
			}
		}

		wg.Wait()

		close(w.GetResults())
		close(w.GetErrors())
	}
}

// Benchmark comparing FIFO vs fixed pool vs dynamic pool on a medium CPU+memory heavy workload.
func BenchmarkFIFO_vs_Pools_Matrix256_64Tasks(b *testing.B) {
	n := 256         // medium CPU and memory per task
	tasksCount := 64 // medium number of tasks
	tasks := make([]workers.Task[float64], 0, tasksCount)
	for i := 0; i < tasksCount; i++ {
		tasks = append(tasks, workers.TaskValue[float64](cpuHeavyTask(n)))
	}

	ctx := context.Background()

	b.Run("fifo", func(b *testing.B) {
		runBench[float64](b, tasks, func() testWorkers[float64] {
			return newFIFO[float64](ctx, &workers.Config{StartImmediately: true})
		})
	})

	b.Run("fixed_NCPU", func(b *testing.B) {
		runBench[float64](b, tasks, func() testWorkers[float64] {
			w, err := workers.NewOptions[float64](ctx, workers.WithFixedPool(uint(runtime.NumCPU())), workers.WithStartImmediately())
			if err != nil {
				b.Fatalf("NewOptions failed: %v", err)
			}
			return w
		})
	})

	b.Run("dynamic", func(b *testing.B) {
		runBench[float64](b, tasks, func() testWorkers[float64] {
			w, err := workers.NewOptions[float64](ctx, workers.WithStartImmediately())
			if err != nil {
				b.Fatalf("NewOptions failed: %v", err)
			}
			return w
		})
	})

	b.Run("dynamic_preserve_order", func(b *testing.B) {
		runBench[float64](b, tasks, func() testWorkers[float64] {
			w, err := workers.NewOptions[float64](ctx, workers.WithStartImmediately(), workers.WithPreserveOrder())
			if err != nil {
				b.Fatalf("NewOptions failed: %v", err)
			}
			return w
		})
	})

	// RunAll variants to measure orchestration overhead vs direct pool usage.
	b.Run("run_all_fixed_NCPU", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := workers.RunAll[float64](
				ctx,
				tasks,
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			)
			if err != nil {
				b.Fatalf("RunAll failed: %v", err)
			}
		}
	})

	b.Run("run_all_dynamic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := workers.RunAll[float64](
				ctx,
				tasks,
				workers.WithDynamicPool(),
				workers.WithStartImmediately(),
			)
			if err != nil {
				b.Fatalf("RunAll failed: %v", err)
			}
		}
	})

	b.Run("run_all_dynamic_preserve_order", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := workers.RunAll[float64](
				ctx,
				tasks,
				workers.WithDynamicPool(),
				workers.WithStartImmediately(),
				workers.WithPreserveOrder(),
			)
			if err != nil {
				b.Fatalf("RunAll failed: %v", err)
			}
		}
	})

	b.Run("mt_NCPU", func(b *testing.B) {
		runBench[float64](b, tasks, func() testWorkers[float64] {
			return newMT[float64](ctx, &workers.Config{MaxWorkers: uint(runtime.NumCPU()), StartImmediately: true})
		})
	})
}

// Benchmark comparing FIFO vs fixed pool vs dynamic pool on a medium memory-heavy workload.
func BenchmarkFIFO_vs_Pools_Alloc4MiB_64Tasks(b *testing.B) {
	size := 4 * 1024 * 1024 // 4 MiB per task
	tasksCount := 64        // medium number of tasks
	tasks := make([]workers.Task[float64], 0, tasksCount)
	for i := 0; i < tasksCount; i++ {
		tasks = append(tasks, workers.TaskValue[float64](memHeavyTask(size)))
	}

	ctx := context.Background()

	b.Run("fifo", func(b *testing.B) {
		runBench[float64](b, tasks, func() testWorkers[float64] {
			return newFIFO[float64](ctx, &workers.Config{StartImmediately: true})
		})
	})

	b.Run("fixed_NCPU", func(b *testing.B) {
		runBench[float64](b, tasks, func() testWorkers[float64] {
			w, err := workers.NewOptions[float64](ctx, workers.WithFixedPool(uint(runtime.NumCPU())), workers.WithStartImmediately())
			if err != nil {
				b.Fatalf("NewOptions failed: %v", err)
			}
			return w
		})
	})

	b.Run("dynamic", func(b *testing.B) {
		runBench[float64](b, tasks, func() testWorkers[float64] {
			w, err := workers.NewOptions[float64](ctx, workers.WithStartImmediately())
			if err != nil {
				b.Fatalf("NewOptions failed: %v", err)
			}
			return w
		})
	})

	b.Run("dynamic_preserve_order", func(b *testing.B) {
		runBench[float64](b, tasks, func() testWorkers[float64] {
			w, err := workers.NewOptions[float64](ctx, workers.WithStartImmediately(), workers.WithPreserveOrder())
			if err != nil {
				b.Fatalf("NewOptions failed: %v", err)
			}
			return w
		})
	})

	// RunAll variants to measure orchestration overhead vs direct pool usage.
	b.Run("run_all_fixed_NCPU", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := workers.RunAll[float64](
				ctx,
				tasks,
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			)
			if err != nil {
				b.Fatalf("RunAll failed: %v", err)
			}
		}
	})

	b.Run("run_all_dynamic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := workers.RunAll[float64](
				ctx,
				tasks,
				workers.WithDynamicPool(),
				workers.WithStartImmediately(),
			)
			if err != nil {
				b.Fatalf("RunAll failed: %v", err)
			}
		}
	})

	b.Run("run_all_dynamic_preserve_order", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := workers.RunAll[float64](
				ctx,
				tasks,
				workers.WithDynamicPool(),
				workers.WithStartImmediately(),
				workers.WithPreserveOrder(),
			)
			if err != nil {
				b.Fatalf("RunAll failed: %v", err)
			}
		}
	})

	b.Run("mt_NCPU", func(b *testing.B) {
		runBench[float64](b, tasks, func() testWorkers[float64] {
			return newMT[float64](ctx, &workers.Config{MaxWorkers: uint(runtime.NumCPU()), StartImmediately: true})
		})
	})
}
