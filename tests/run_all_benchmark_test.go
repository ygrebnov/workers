package tests

import (
	"context"
	"runtime"
	"testing"

	"github.com/ygrebnov/workers"
)

// BenchmarkRunAll provides a small sanity benchmark for RunAll itself.
// Main overhead comparisons are in fifo_vs_pools_benchmark_test.go.
func BenchmarkRunAll(b *testing.B) {
	ctx := context.Background()
	// Reuse helper from benchmark_test.go via same package.
	tasks := getTasks(1_000, 50_000, 1_000)

	b.Run("dynamic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := workers.RunAll[string](
				ctx,
				tasks,
				workers.WithDynamicPool(),
				workers.WithStartImmediately(),
			)
			if err != nil {
				b.Fatalf("RunAll dynamic failed: %v", err)
			}
		}
	})

	b.Run("fixed_NCPU", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := workers.RunAll[string](
				ctx,
				tasks,
				workers.WithFixedPool(uint(runtime.NumCPU())),
				workers.WithStartImmediately(),
			)
			if err != nil {
				b.Fatalf("RunAll fixed failed: %v", err)
			}
		}
	})
}
