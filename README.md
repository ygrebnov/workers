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
w := workers.NewOptions[string](
    context.Background(),
    workers.WithDynamicPool(),
    workers.WithStartImmediately(),
    workers.WithTasksBuffer(16),
    workers.WithResultsBuffer(2048),
    workers.WithErrorsBuffer(2048),
)
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
	"sync"

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

	wg := sync.WaitGroup{}

	// Receive and print results or handle errors in a separate goroutine.
	go func() {
		for range 10 {
			select {
			case result := <-w.GetResults():
				fmt.Println(result)
			case err := <-w.GetErrors():
				fmt.Println("error executing task:", err)
			}

			wg.Done()
		}
	}()

	for i := 20; i >= 11; i-- {
		wg.Add(1)

		// Add ten tasks calculating the Fibonacci number for a given index.
		// A task is a function that takes a context and returns a string.
		err := w.AddTask(func(ctx context.Context) string {
			return fmt.Sprintf("Calculated Fibonacci for: %d, result: %d.", i, fibonacci(i))
		})
		if err != nil {
			log.Fatalf("failed to add task: %v", err)
		}
	}

	wg.Wait() // Wait for all tasks to finish.

	// Close channels.
	close(w.GetResults())
	close(w.GetErrors())
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

## Contributing

Contributions are welcome!  
Please open an [issue](https://github.com/ygrebnov/workers/issues) or submit a [pull request](https://github.com/ygrebnov/workers/pulls).

## License

Distributed under the MIT License. See the [LICENSE](LICENSE) file for details.