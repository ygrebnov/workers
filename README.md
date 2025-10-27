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

## Constructor

The primary constructor uses functional options to configure the `Workers` instance:

- `New(ctx context.Context, opts ...Option) (*Workers[R], error)`

Example:
```go
w, err := workers.New[string](
    context.Background(),
    workers.WithStartImmediately(),
    workers.WithTasksBuffer(16),
)
if err != nil {
    panic(err)
}
```

## Zero-Value Usage
- The zero value of `Workers` is usable. Call `Start(ctx)` to initialize it with default settings.
- For custom configuration, use the `New` constructor with options.

## Defaults

Unless overridden via options, a new instance uses:

- **Pool:** Dynamic-size pool
- **StartImmediately:** `false` (explicit `Start()` call is required)
- **StopOnError:** `false`
- **TasksBufferSize:** 0 (unbuffered)
- **ResultsBufferSize:** 1024
- **ErrorsBufferSize:** 1024
- **StopOnErrorErrorsBufferSize:** 100

## Usage Example

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
	// Create a new controller with a dynamic number of workers that starts immediately.
	// The type parameter specifies the type of the task result.
	w, err := workers.New[string](context.Background(), workers.WithStartImmediately())
	if err != nil {
		log.Fatalf("failed to create workers: %v", err)
	}

	// Enqueue ten tasks calculating the Fibonacci number for a given index.
	for i := 20; i >= 11; i-- {
		val := i // Capture loop variable
		err := w.AddTask(workers.TaskValue[string](func(ctx context.Context) string {
			return fmt.Sprintf("Calculated Fibonacci for: %d, result: %d.", val, fibonacci(val))
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
			if !ok {
				resultsClosed = true
				continue
			}
			fmt.Println(r)
		case err, ok := <-w.GetErrors():
			if !ok {
				errorsClosed = true
				continue
			}
			fmt.Println("error executing task:", err)
		}
	}
}
```

---

### Example Output

```text
Calculated Fibonacci for: 20, result: 6765.
Calculated Fibonacci for: 18, result: 2584.
Calculated Fibonacci for: 19, result: 4181.
...
Calculated Fibonacci for: 11, result: 89.
```

## Helpers

These top-level helpers make common usage patterns concise while preserving the core `Workers` semantics (pool options, `StopOnError`, `PreserveOrder`, buffers).

- **RunAll**
    - `RunAll[R any](ctx context.Context, tasks []Task[R], opts ...Option) ([]R, error)`
    - Batch-executes `Task[R]`, owns the lifecycle (Start → enqueue → wait → Close), and returns all results with an aggregated error (`errors.Join`).
    - **Ordering:** Completion order by default; `WithPreserveOrder` emits results in input order.
    - **StopOnError:** Cancels on the first error; tasks not yet started may not run.

- **Map**
    - `Map[T any, R any](ctx context.Context, in []T, fn func(context.Context, T) (R, error), opts ...Option) ([]R, error)`
    - A convenience wrapper over `RunAll` for mapping a slice through a function that returns `(R, error)`.

- **ForEach**
    - `ForEach[T any](ctx context.Context, in []T, fn func(context.Context, T) error, opts ...Option) error`
    - Applies a side-effecting function to each input and aggregates errors, with no results channel.

- **RunStream**
    - `RunStream[R any](ctx context.Context, in <-chan Task[R], opts ...Option) (<-chan R, <-chan error, error)`
    - Consumes `Task[R]` from a channel and forwards results and errors via returned channels; owns the lifecycle.
    - **Ordering:** Completion order by default; `WithPreserveOrder` enforces input order.
    - **StopOnError:** Cancels on the first error; the forwarder stops reading from the input channel.

- **MapStream**
    - `MapStream[T any, R any](ctx context.Context, in <-chan T, fn func(context.Context, T) (R, error), opts ...Option) (<-chan R, <-chan error, error)`
    - A convenience wrapper over `RunStream` that wraps each `T` from the input channel into a `Task[R]` internally.

- **ForEachStream**
    - `ForEachStream[T any](ctx context.Context, in <-chan T, fn func(context.Context, T) error, opts ...Option) (<-chan error, error)`
    - Applies a side-effecting function to each streamed input and returns an errors channel. The channel closes when the stream completes or is canceled.

**Notes on Helpers:**
- **Backpressure:** Stream helpers propagate backpressure via configured buffers and by requiring consumers to drain the returned channels.
- **Cancellation:** With `StopOnError`, the internal controller context is canceled on the first error. Stream forwarders stop reading from the input channel and wait for already-started tasks to finish before closing the output channels.

## AddTask semantics (no-panic backpressure)

- Concurrency: AddTask is safe for concurrent use by multiple goroutines.
- After Start():
    - If the internal context is canceled (Close or StopOnError), AddTask fails fast with ErrInvalidState.
    - Otherwise, AddTask enqueues the task and may block while the tasks channel is full; if cancellation happens while blocked, the call unblocks and returns ErrInvalidState.
- Before Start():
    - If TasksBufferSize > 0, AddTask enqueues into the buffer and may block when the buffer is full.
    - If TasksBufferSize == 0, AddTask returns ErrInvalidState because there is nowhere to put the task yet.
- No panics: AddTask never panics due to queue saturation.

## Channels and `Close`
- The library owns the `Results` and `Errors` channels. When you call `Close()`, it:
    1. Cancels the internal context to stop dispatching new work.
    2. Waits for all in-flight tasks to finish.
    3. Forwards any buffered internal errors (in `StopOnError` mode) on a best-effort basis.
    4. Closes both channels.
- **Do not close the channels yourself if you use `Close()`**; Go panics on a double-close.
- **Advanced:** If you manage the lifecycle manually and close channels yourself, do not call `Close()`.

### `StopOnError` (Cancel-First) Notes
- With `WithStopOnError()`:
    - On the first error, the controller cancels promptly to stop scheduling new work.
    - The triggering error is forwarded to the outward `Errors` channel. If the channel is full (i.e., no reader), it’s forwarded via a detached goroutine and delivered once a reader appears. If `Close()` happens first, pending errors may be dropped.

### `StopOnError` Forwarding Semantics (Buffered vs. Unbuffered)

This note documents how `StopOnError` forwards the first error and cancels scheduling, and how the outward error channel's buffering changes behavior.

#### Overview
When `WithStopOnError()` is enabled:
- On the first error from any worker, the controller cancels promptly (cancel-first), which stops further task scheduling and lets in-flight tasks observe `ctx.Done()`.
- The triggering error is then forwarded to the outward errors channel.
- If the outward channel cannot accept the error immediately, the controller uses a detached goroutine to deliver the error when a reader eventually appears. If `Close()` occurs first, the pending detached send is signaled to exit, and the error may be dropped (best-effort delivery).
- After cancellation, `AddTask` returns `ErrInvalidState` deterministically and never blocks.

#### Internal vs. Outward Buffers
- **Internal Buffer (Producer Side):**
    - Size is controlled by `WithStopOnErrorBuffer(size)`.
    - Workers push their errors into this internal buffer when `StopOnError` is enabled.
    - The controller consumes from this buffer, cancels first, and only then forwards the error outward.
- **Outward Buffer (Consumer Side):**
    - Size is controlled by `WithErrorsBuffer(size)`.
    - It determines how the controller forwards errors to consumers of `GetErrors()`.

#### Buffered Outward Errors (`size > 0`)
- Cancel-first happens immediately.
- The first error is forwarded synchronously if there is capacity in the outward buffer.
- If the outward channel has room, forwarding is non-blocking and immediate.

**Example:**
- Options: `WithStopOnError()`, `WithStopOnErrorBuffer(1)`, `WithErrorsBuffer(1)`
- Behavior: Cancellation occurs, and the error is forwarded synchronously into the outward channel if it has capacity.

#### Unbuffered or Saturated Outward Errors (`size == 0`, or no reader)
- Cancel-first happens immediately.
- Forwarding is performed via a detached goroutine that delivers the error when a reader appears.
- If `Close()` happens before a reader is ready, the pending detached send is signaled to exit, and the error may be dropped (best-effort delivery).

**Example:**
- Options: `WithStopOnError()`, `WithStopOnErrorBuffer(1)`, `WithErrorsBuffer(0)`
- Behavior: Cancellation occurs; the error will be delivered once a reader starts receiving from `GetErrors()`.

#### `Close` Semantics with `StopOnError`
`Close()`:
- Cancels the internal context (which stops the dispatcher and the `StopOnError` forwarder).
- Waits for in-flight tasks to finish.
- Waits for the forwarder to stop, then signals and waits for any detached senders.
- Drains any remaining internal errors into the outward channel on a best-effort (non-blocking) basis, then closes the results and errors channels.

#### Guidance
- Prefer a buffered outward errors channel when you want the first error to be forwarded synchronously without requiring an immediate reader.
- Use an unbuffered outward errors channel when you need strict backpressure (readers must be present to receive). Know that delivery is best-effort if `Close()` happens before a reader appears.
- Keep the `StopOnError` internal buffer size small to propagate cancellation quickly under error bursts.

---

## Preserve Order (`WithPreserveOrder`)

Enable deterministic, input-order delivery of results by adding `workers.WithPreserveOrder()` when constructing `Workers`. When enabled, the outward results channel emits results in the same order as tasks were added (by input index), not by completion time.

### Usage

```go
w, err := workers.New[int](
    ctx,
    workers.WithStartImmediately(),
    workers.WithPreserveOrder(),
)
if err != nil {
    panic(err)
}

// Add mixed-duration tasks; results will be emitted by input index.
_ = w.AddTask(workers.TaskValue[int](func(ctx context.Context) int { /* ... */ return 1 }))
// ...
```

This option applies to all APIs that use the core `Workers` controller, including helpers like `RunAll` and `Map`.

### What is Considered a "Result"
- A task contributes a result only if it returns `err == nil` and was created to send a result (e.g., via `TaskValue` or a `TaskFunc`).
- Tasks created with `TaskError` or tasks that return an error do not emit a value to the results channel. They still generate an internal completion event so that ordering can advance past their index.

### How It Works (High-Level)
- Each task is assigned an input index on `AddTask`.
- Workers emit a completion event for every executed task.
- A coordinator buffers out-of-order completions and only emits to `GetResults()` when the result for the next expected index is available. It skips indices for tasks that produced no result.

### Trade-offs
- **Head-of-Line Blocking:** If task `i` is slow, results for tasks `i+1`, `i+2`, etc., are held until `i` completes. This can reduce throughput if task completion times are skewed.
- **Buffering and Memory:** Out-of-order completions are buffered. Memory usage grows with the number of completed but not-yet-emitted results. When buffers fill, workers block on sending completion events, applying backpressure.
- **Extra Coordination:** One additional goroutine coordinates reordering when enabled. When disabled, there is no overhead.

### Interaction with `StopOnError`
- **Cancel-First:** On the first error, `WithStopOnError()` cancels scheduling. Later tasks may never start and thus never produce a completion event.
- **Contiguous Prefix Only:** Because ordering must advance index-by-index, the coordinator can only emit a contiguous prefix of results up to the first index for which it never receives a completion event. Results after that point are not emitted.
- **Error Delivery:** Errors are still sent on `GetErrors()` per normal `StopOnError` semantics.

### Guidance
- Use `WithPreserveOrder` when consumers require input-order determinism (e.g., mapping inputs to outputs by position).
- Prefer leaving it disabled when fastest-first delivery is desirable.
- Tune `ResultsBufferSize` to balance memory and throughput under skew; larger buffers allow more out-of-order completions before backpressure engages.

---

## Choosing Between Dynamic and Fixed-Size Pools

### When a Dynamic-Size Pool is a Better Fit
- **Bursty or unpredictable workloads:** The pool grows during spikes to keep latency down, then shrinks.
- **Mostly I/O-bound tasks:** For network, disk, or DB calls where goroutines mostly wait, higher concurrency often improves throughput.
- **Heterogeneous task durations:** A mix of long and short tasks; the pool can add workers so short tasks aren’t delayed by long ones.
- **Many small, lightweight tasks:** Avoids careful capacity tuning.

**Caveats with Dynamic Pools:**
- **Resource Safety:** The dynamic pool is effectively unbounded. Add an explicit semaphore or rate limiter when calling bounded backends (DBs, APIs).
- **CPU-Heavy Tasks:** Unbounded concurrency can oversubscribe CPUs. Prefer a fixed pool sized near `runtime.NumCPU()`.

### When a Fixed-Size Pool is Preferable
- **CPU-bound or compute-heavy work:** Bound concurrency to `~runtime.NumCPU()` for best throughput and lower contention.
- **Hard external limits:** For DB connection pools, API rate limits, or other bounded systems, a fixed pool provides predictable backpressure.
- **Need for a strict cap or predictable SLAs:** A fixed pool gives a hard limit on concurrent work.
- **Long-lived or expensive per-worker state:** Reliably retain and reuse initialized workers.

### Quick Decision Matrix
- **I/O-bound, unknown concurrency:** Dynamic-size pool (default). Add a limiter if you talk to bounded backends.
- **CPU-bound compute:** Fixed-size pool with `MaxWorkers ≈ runtime.NumCPU()`.
- **Bursty workload where peak latency matters:** Dynamic-size pool.
- **Integrating with a bounded backend (DB/API):** Fixed-size pool sized to the backend's limits (or dynamic + explicit semaphore).

### Snippets

**Dynamic Pool + Explicit Limiter for I/O-Bound Tasks**
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
    w, err := workers.New[error](
        ctx,
        workers.WithStartImmediately(),
    )
    if err != nil {
        panic(err)
    }

    _ = w.AddTask(workers.TaskError[error](func(ctx context.Context) error {
        sem <- struct{}{}            // Acquire
        defer func() { <-sem }()     // Release
        // Do I/O-bound work (HTTP/DB/etc.).
        return nil
    }))
}
```

**Fixed-Size Pool for CPU-Bound Tasks**
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
    w, err := workers.New[int](
        ctx,
        workers.WithFixedPool(n),
        workers.WithStartImmediately(),
    )
    if err != nil {
        panic(err)
    }

    _ = w.AddTask(workers.TaskValue[int](func(ctx context.Context) int {
        return cpuHeavy(42)
    }))
}
```

## WithIntakeChannel: user-supplied intake (exclusive mode)

When you already have a pipeline stage that produces `Task[R]` values (or you want full control over buffering/fan-in/closing), you can configure Workers to read tasks exclusively from a channel you own.

- API: `WithIntakeChannel[R any](in <-chan Task[R]) Option`
- Ownership: you send tasks into `in` and close it when done.
- Exclusive mode: while an intake channel is configured, `AddTask`/`AddTaskContext`/`TryAddTask` return `ErrExclusiveIntakeChannel`.
- Pre-start sends: you can send before `Start()`; values will sit in your channel’s buffer and will be drained after `Start()`.
- Cancellation: after `Close()` or on `WithStopOnError`, Workers stop consuming; senders may block on `in` depending on its capacity—design your pipeline accordingly.
- Type validation: `New[R]` validates that `in` is a `(<-chan Task[R])`; a mismatch returns `ErrInvalidConfig`.

### When to use
- You already have task producers and want Workers to simply drain a channel.
- You need to choose your own buffering/fan-in and close semantics.
- You want to enqueue before `Start()` without using the internal tasks buffer.

### Example
```go
package main

import (
	"context"
	"log"
	"runtime"
	"time"

	"github.com/ygrebnov/workers"
)

func main() {
	ctx := context.Background()

	// 1) Create the intake channel you own. Use any buffer size you need.
	in := make(chan workers.Task[int], 8)

	// 2) Construct Workers and start immediately.
	w, err := workers.New[int](
		ctx,
		workers.WithIntakeChannel(in),
		workers.WithFixedPool(uint(runtime.NumCPU())),
		workers.WithStartImmediately(),
	)
	if err != nil {
		log.Fatalf("setup failed: %v", err)
	}

	// 3) Produce tasks and close the channel when done.
	go func() {
		for i := 0; i < 5; i++ {
			v := i
			in <- workers.TaskValue[int](func(context.Context) int {
				time.Sleep(5 * time.Millisecond)
				return v * 2
			})
		}
		close(in)
	}()

	// 4) Consume results and errors until closed (typical pattern).
	for {
		select {
		case r, ok := <-w.GetResults():
			if !ok {
				w.Close() // ensure cleanup if you break out here
				return
			}
			log.Printf("result=%d", r)
		case err, ok := <-w.GetErrors():
			if !ok {
				// no more errors
				continue
			}
			log.Printf("error=%v", err)
		}
	}
}
```

Notes:
- You can pass a bidirectional channel `chan Task[R]`; it implicitly satisfies `<-chan Task[R]`.
- Don’t use `AddTask`/`AddTaskContext`/`TryAddTask` while `WithIntakeChannel` is configured—they return `ErrExclusiveIntakeChannel` by design.

### Interactions with options
- Preserve-order (`WithPreserveOrder`): input indices are assigned when tasks are admitted from `in`; results are emitted in input order.
- Error tagging (`WithErrorTagging`): any error from a task is wrapped with task ID and input index metadata.
- Stop-on-error (`WithStopOnError`): the first error cancels the Workers controller; the intake-forwarder stops reading `in`. Your producers may block if they keep sending; choose an appropriate buffer or select on context/timeouts.

### Common pitfalls
- Forgetting to close `in`: the intake-forwarder exits on `in` close or cancellation; if you never close it and never cancel, the forwarder goroutine will keep waiting for more tasks.
- Blocking producers on cancel: after cancellation (stop-on-error or `Close()`), Workers stop draining `in`; producers that keep sending on a small/unbuffered channel will block. Prefer buffering and/or select with a done context.
- Type mismatch: `WithIntakeChannel` must match `R` from `New[R]`. If you pass `chan Task[string]` to `Workers[int]`, `New` will return `ErrInvalidConfig` with a clear message.

### Error semantics recap
- `ErrExclusiveIntakeChannel`: returned by `AddTask`/`AddTaskContext`/`TryAddTask` when an intake channel is configured.
- `ErrInvalidConfig`: returned by `New` if the intake channel element type doesn’t match `R`.

## Contributing

Contributions are welcome!  
Please open an [issue](https://github.com/ygrebnov/workers/issues) or submit a [pull request](https://github.com/ygrebnov/workers/pulls).

## License

Distributed under the MIT License. See the [LICENSE](LICENSE) file for details.
