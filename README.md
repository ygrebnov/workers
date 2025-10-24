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

## Zero-value usage
- The zero value of Workers is usable after calling Start(ctx); it initializes with the same defaults listed below.
- For custom configuration, construct with New or NewOptions.

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

## Helpers

These top-level helpers make common usage patterns concise while preserving the core Workers semantics (pool options, StopOnError, preserve-order, buffers).

- RunAll
    - Signature: RunAll[R any](ctx context.Context, tasks []Task[R], opts ...Option) ([]R, error)
    - Batch-executes Task[R], owns lifecycle (Start → enqueue → wait → Close), returns all results and an aggregated error (errors.Join).
    - Ordering: completion order by default; WithPreserveOrder emits results in input order.
    - StopOnError: cancels on first error; tasks not yet started may not run.

- Map
    - Signature: Map[T any, R any](ctx context.Context, in []T, fn func(context.Context, T) (R, error), opts ...Option) ([]R, error)
    - Convenience over RunAll for mapping a slice through a function that returns (R, error).

- ForEach
    - Signature: ForEach[T any](ctx context.Context, in []T, fn func(context.Context, T) error, opts ...Option) error
    - Applies a side-effecting function to each input, aggregates errors, no results channel.

- RunStream
    - Signature: RunStream[R any](ctx context.Context, in <-chan Task[R], opts ...Option) (<-chan R, <-chan error, error)
    - Consumes Task[R] from a channel, forwards results and errors via returned channels; owns lifecycle.
    - Ordering: completion order by default; WithPreserveOrder enforces input order.
    - StopOnError: cancels on first error; forwarder stops reading input.

- MapStream
    - Signature: MapStream[T any, R any](ctx context.Context, in <-chan T, fn func(context.Context, T) (R, error), opts ...Option) (<-chan R, <-chan error, error)
    - Convenience over RunStream that wraps each T into a Task[R] internally.

- ForEachStream
    - Signature: ForEachStream[T any](ctx context.Context, in <-chan T, fn func(context.Context, T) error, opts ...Option) (<-chan error, error)
    - Applies a side-effecting function to each streamed input and returns an errors channel; errs closes when stream completes or is canceled.

Notes
- Backpressure: stream helpers propagate backpressure via configured buffers and by requiring consumers to drain the returned channels.
- Cancellation: with StopOnError, the internal controller context is canceled on the first error; stream forwarders stop reading input and wait for already-started tasks before closing channels.

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

### StopOnError forwarding semantics (buffered vs unbuffered)

This note documents how StopOnError forwards the first error and cancels scheduling, and how outward error channel buffering changes behavior.

#### Overview
When `WithStopOnError()` is enabled:
- On the first error produced by any worker, the controller cancels promptly (cancel-first), which stops further task scheduling and lets in-flight tasks observe `ctx.Done()`.
- The triggering error is then forwarded to the outward errors channel.
- If the outward channel cannot accept the error immediately, the controller uses a detached goroutine to deliver the error when a reader eventually appears. If `Close()` occurs first, the pending detached send is signaled to exit and the error may be dropped (best-effort delivery).
- After cancellation, `AddTask` returns `ErrInvalidState` deterministically and never blocks.

#### Internal vs outward buffers
- Internal buffer (producer side):
    - Size is controlled by `WithStopOnErrorBuffer(size)` / `StopOnErrorErrorsBufferSize`.
    - Workers push their errors into this internal buffer when StopOnError is enabled.
    - The controller consumes from this buffer, cancels first, and only then forwards outward.
- Outward buffer (consumer side):
    - Size is controlled by `WithErrorsBuffer(size)` / `ErrorsBufferSize`.
    - It determines how the controller forwards errors to consumers of `GetErrors()`.

#### Buffered outward errors (size > 0)
- Cancel-first happens immediately.
- The first error is forwarded synchronously if there is capacity left in the outward buffer.
- If the outward channel has room, forwarding is non-blocking and immediate.

Example:
- Options: `WithStopOnError()`, `WithStopOnErrorBuffer(1)`, `WithErrorsBuffer(1)`
- Behavior: cancellation occurs, the error is forwarded synchronously into the outward channel if it has capacity.

#### Unbuffered or saturated outward errors (size == 0, or no reader yet)
- Cancel-first happens immediately.
- Forwarding is performed via a detached goroutine that delivers the error when a reader appears.
- If `Close()` happens before a reader is ready, the pending detached send is signaled to exit; the error may be dropped (best-effort delivery).

Example:
- Options: `WithStopOnError()`, `WithStopOnErrorBuffer(1)`, `WithErrorsBuffer(0)`
- Behavior: cancellation occurs; the error will be delivered once a reader starts receiving from `GetErrors()`.

#### Close semantics with StopOnError
`Close()`:
- Cancels the internal context (stops dispatch and the StopOnError forwarder).
- Waits for in-flight tasks to finish.
- Waits for the forwarder to stop, then signals and waits for any detached senders.
- Drains any remaining internal errors best-effort into the outward channel (non-blocking), then closes results and errors channels.

#### Guidance
- Prefer a buffered outward errors channel when you want the first error to be forwarded synchronously without requiring an immediate reader.
- Use an unbuffered outward errors channel when you need strict backpressure (readers must be present to receive). Know that delivery is best-effort if `Close()` happens before a reader appears.
- Keep `StopOnErrorErrorsBufferSize` small to propagate cancellation quickly under error bursts; increase only if you expect brief spikes before a reader drains.

---

## Preserve order (WithPreserveOrder)

Enable deterministic, input-order delivery of results by adding `workers.WithPreserveOrder()` when constructing Workers. When enabled, the outward results channel emits results in the same order as tasks were added (by input index), not by completion time.

Implementation details are sketched in `improvements/preserve_order.md#Implementation` and realized in the core via a coordinator goroutine and completion events; see that doc for a deeper technical design.

### Usage

```go
w, err := workers.NewOptions[int](
    ctx,
    workers.WithDynamicPool(),
    workers.WithStartImmediately(),
    workers.WithPreserveOrder(),
)
if err != nil { panic(err) }

// Add mixed-duration tasks in any order; results will be emitted by input index.
_ = w.AddTask(workers.TaskValue[int](func(ctx context.Context) int { /* ... */ return 1 }))
// ...
```

Applies to all APIs using the core Workers controller (including helpers like `RunAll` and `Map` that forward options).

### What is considered a "result"
- A task contributes a result only if it returns `err == nil` and `SendResult() == true` (i.e., created via `TaskValue` or a `TaskFunc` that signals result emission).
- Tasks created with `TaskError` (no outward result) or tasks that return an error do not emit a value to the results channel. They still generate an internal completion event so ordering can advance past their index without emitting a value.

### How it works (high level)
- Each task is assigned an input index on `AddTask`.
- Workers emit a completion event for every executed task:
    - present=true with the value when the task succeeded and wants to send a result.
    - present=false when the task failed or is a no-result task.
- A coordinator buffers out-of-order completions and only emits to `GetResults()` when the next index is available; it skips indices with `present=false`.

### Trade-offs
- Head-of-line blocking: if task i is slow and completes after task i+K, results for indices > i are held until i completes (or is seen as no-result/error). This can reduce throughput when completion times are skewed.
- Buffering and memory: out-of-order completions are buffered. The internal events channel is sized by `ResultsBufferSize`. Under skew, memory usage grows with the number of completed but not-yet-emittable results. When buffers fill, workers block on sending completion events, applying backpressure to execution.
- Extra coordination: one additional goroutine coordinates reordering when enabled. When disabled, there is no coordination overhead.

### Interaction with StopOnError
- Cancel-first: on the first error, `WithStopOnError()` cancels scheduling. Some later tasks may never start and therefore never produce a completion event.
- Contiguous prefix only: because ordering must advance index-by-index, the coordinator can only emit a contiguous prefix of results up to the first index for which it never receives a completion event (e.g., a never-started task). Results after that point are not emitted, and the results channel closes after cleanup.
- Error delivery: errors are still sent on `GetErrors()` per normal StopOnError semantics. A task that both intended to send a result and returned an error does not emit a result; the coordinator advances past its index due to the completion event with `present=false`.

### Interaction with no-result tasks
- Tasks created with `TaskError` (or any task where `SendResult()==false`) still participate in ordering by emitting a completion event with `present=false`. The coordinator immediately advances past those indices without emitting a value.
- This means ordering is defined over the subset of tasks that actually produce results. You get values in input order for successful, result-producing tasks; no-result tasks simply create "gaps" that are skipped.

### Guidance
- Use `WithPreserveOrder` when consumers require input-order determinism (e.g., mapping inputs to outputs by position).
- Prefer leaving it disabled when fastest-first delivery is desirable (e.g., streaming pipelines that benefit from early results).
- Tune `ResultsBufferSize` to balance memory and throughput under skew; larger buffers allow more in-flight out-of-order completions before backpressure engages.

### Close and lifecycle
- `Close()` waits for in-flight tasks to finish, then closes the internal events stream and the coordinator drains any remaining contiguous tail before exiting, after which results and errors channels are closed.
- In StopOnError scenarios, because some indices may never produce events, only the contiguous prefix of results will be observed before channels close.

---

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
