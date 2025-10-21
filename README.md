[![Workers](https://yaroslavgrebnov.com/assets/images/workers_lightweight_go_library_for_executing_multiple_tasks_concurrently-4b69e8ac142599731c323ca71483a879.png)](https://yaroslavgrebnov.com/projects/workers/overview)

**Workers** is a **lightweight Go library** for executing **multiple tasks concurrently**, with support for either a dynamic or fixed number of workers. Written by [ygrebnov](https://github.com/ygrebnov).

Designed to be **simple and easy to use**, it allows executing tasks concurrently with minimal setup. The library is suitable for a variety of use cases, from simple parallel processing to more complex workflows.

---

[![GoDoc](https://pkg.go.dev/badge/github.com/ygrebnov/workers)](https://pkg.go.dev/github.com/ygrebnov/workers)
[![Build Status](https://github.com/ygrebnov/workers/actions/workflows/build.yml/badge.svg)](https://github.com/ygrebnov/workers/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/ygrebnov/workers/graph/badge.svg?token=1TY5NH8IF6)](https://codecov.io/gh/ygrebnov/workers)
[![Go Report Card](https://goreportcard.com/badge/github.com/ygrebnov/workers)](https://goreportcard.com/report/github.com/ygrebnov/workers)

[User Guide](https://yaroslavgrebnov.com/projects/workers/overview) | [Examples](https://yaroslavgrebnov.com/projects/workers/examples) | [Contributing](#contributing)

## Features

- **Flexible Worker Pools:** Execute tasks concurrently using either dynamic (scalable) or fixed-size worker pools.
- **Streaming Results and Errors:** Task results and errors are provided via channels.
- **Configurable Error Handling:** Option to stop immediately upon encountering an error or continue execution.
- **Robust Panic Recovery:** Safely handles panics within tasks, preventing crashes.
- **Delayed Task Execution:** Tasks can be accumulated first and executed later on-demand.

## Installation

Requires Go 1.22 or later:

```shell
go get github.com/ygrebnov/workers
```

## Constructors

- NewOptions(ctx, opts ...Option): preferred constructor using functional options.
- New(ctx, *Config): current stable constructor; planned for deprecation in a future major version.

## Defaults

Unless overridden, a new instance uses:

- MaxWorkers: 0 (dynamic pool)
- StartImmediately: false (call Start() if TasksBufferSize == 0)
- StopOnError: false
- TasksBufferSize: 0
- ResultsBufferSize: 1024
- ErrorsBufferSize: 1024
- StopOnErrorErrorsBufferSize: 100

## Options vs Config

Options-based (recommended):

```go
w, err := workers.NewOptions[string](
    context.Background(),
    workers.WithDynamicPool(),
    workers.WithStartImmediately(),
    workers.WithTasksBuffer(16),
    workers.WithResultsBuffer(2048),
    workers.WithErrorsBuffer(2048),
)
if err != nil { panic(err) }
```

Equivalent with Config:

```go
w := workers.New[string](
    context.Background(),
    &workers.Config{
        // dynamic pool (MaxWorkers=0)
        StartImmediately:  true,
        TasksBufferSize:   16,
        ResultsBufferSize: 2048,
        ErrorsBufferSize:  2048,
    },
)
```

## Usage example

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ygrebnov/workers"
)

// fibonacci calculates Fibonacci numbers.
func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

func main() {
	// Create a new controller with a dynamic number of workers.
	// The controller starts immediately.
	// Type parameter is used to specify the type of task result.
	w := workers.New[string](context.Background(), &workers.Config{StartImmediately: true})

	// Enqueue ten tasks calculating the Fibonacci number for a given index.
	for i := 20; i >= 11; i-- {
		err := w.AddTask(workers.TaskValue[string](func(ctx context.Context) string {
			return fmt.Sprintf("Calculated Fibonacci for: %d, result: %d.", i, fibonacci(i))
		}))
		if err != nil {
			log.Fatalf("failed to add task: %v", err)
		}
	}

	// Close stops scheduling, waits for in-flight tasks to complete, and then
	// closes both results and errors channels owned by this instance.
	w.Close()

	// Drain results and errors until both channels are closed.
	resultsClosed, errorsClosed := false, false
	for !(resultsClosed && errorsClosed) {
		select {
		case r, ok := <-w.GetResults():
			if !ok { resultsClosed = true; continue }
			fmt.Println(r)
		case err, ok := <-w.GetErrors():
			if !ok { errorsClosed = true; continue }
			fmt.Println("error executing task:", err)
		}
	}
}
```

---

Example Output

```text
Calculated Fibonacci for: 20, result: 6765.
Calculated Fibonacci for: 18, result: 2584.
Calculated Fibonacci for: 19, result: 4181.
...
Calculated Fibonacci for: 11, result: 89.
```

## Channels and Close
- The library owns the Results and Errors channels. When you call Close(), it:
  - cancels the internal context so no new work is dispatched,
  - waits for in-flight tasks to finish,
  - forwards any buffered internal errors (StopOnError) best-effort, and
  - closes both channels.
- Don’t close the channels yourself if you use Close(); Go panics on double-close.
- Advanced: if you manage lifecycle manually and close channels yourself, do not call Close().

### Stop-on-error (cancel-first) notes
- With WithStopOnError:
  - On the first error, the controller cancels promptly to stop scheduling new work.
  - The triggering error is forwarded to the outward Errors channel. If the channel is full/no reader,
    it’s forwarded via a detached goroutine and delivered once a reader appears. If Close() happens first,
    pending errors may be dropped.

## Choosing between dynamic and fixed-size pool
### When a dynamic-size pool is a better fit
- Bursty or unpredictable workloads: dynamic grows parallelism during spikes to keep latency down, then shrinks afterward.
- Mostly I/O-bound tasks: network/disk/DB/RPC where goroutines mostly wait; higher concurrency often improves throughput.
- Heterogeneous task durations: mix of long and short tasks; dynamic can add workers so short tasks aren’t delayed by long ones.
- Many small, lightweight tasks: avoids careful capacity tuning; per-worker overhead is low.
- Unknown ideal concurrency: start dynamic, measure, then tune if needed.
  Caveats with dynamic
- Resource safety: dynamic here is effectively unbounded. Add an explicit semaphore/rate limiter when calling bounded backends (DB/APIs).
- CPU-heavy tasks: unbounded concurrency can oversubscribe CPUs; prefer a fixed pool near runtime.NumCPU.
- Expensive per-worker state: sync.Pool may drop items on GC; if workers hold costly state, a capped fixed pool is more predictable for reuse.

### When a fixed-size pool is preferable
- CPU-bound or compute-heavy work: bound concurrency to ~runtime.NumCPU for best throughput and lower contention.
- Hard external limits: DB connection pools, API rate limits, or other bounded systems; fixed provides predictable backpressure.
- Need strict cap/predictable SLAs: fixed gives a hard limit on concurrent work and a clear queue story.
- Long-lived or expensive per-worker state: retain and reuse initialized workers reliably.
  Quick decision matrix
- I/O-bound and don’t know concurrency yet: dynamic-size pool (default). Add a limiter if you talk to bounded backends.
- CPU-bound math/heavy memory compute: fixed-size pool with MaxWorkers ≈ runtime.NumCPU().
- Bursty workload where latency during peaks matters: dynamic-size pool.
- Integrating with a bounded backend (DB/API): fixed-size pool sized to backend limits (or dynamic + explicit semaphore with the same cap).
  
### Snippets

Dynamic + explicit limiter for I/O-bound tasks
```go
package main

import (
    "context"
    "github.com/ygrebnov/workers"
)

func main() {
    ctx := context.Background()
    // Limit calls to an external backend to at most 50 in flight.
    sem := make(chan struct{}, 50)
    w, err := workers.NewOptions[error](
        ctx,
        workers.WithDynamicPool(),
        workers.WithStartImmediately(),
    )
    if err != nil { panic(err) }

    _ = w.AddTask(workers.TaskError[error](func(ctx context.Context) error {
        sem <- struct{}{}            // acquire
        defer func() { <-sem }()     // release
        // Do I/O-bound work (HTTP/DB/etc.).
        return nil
    }))
}
```

Fixed-size pool for CPU-bound tasks
```go
package main

import (
    "context"
    "runtime"
    
    "github.com/ygrebnov/workers"
)

func cpuHeavy(n int) int { /* compute */ return n }
func main() {
    ctx := context.Background()
    n := uint(runtime.NumCPU())
    w, err := workers.NewOptions[int](
        ctx,
        workers.WithFixedPool(n),
        workers.WithStartImmediately(),
    )
    if err != nil { panic(err) }

    _ = w.AddTask(workers.TaskValue[int](func(ctx context.Context) int {
        return cpuHeavy(42)
    }))
}
```

## Contributing

Contributions are welcome!  
Please open an [issue](https://github.com/ygrebnov/workers/issues) or submit a [pull request](https://github.com/ygrebnov/workers/pulls).

## License

Distributed under the MIT License. See the [LICENSE](LICENSE) file for details.
