[![Workers](https://yaroslavgrebnov.com/assets/images/workers_lightweight_go_library_for_executing_multiple_tasks_concurrently-4b69e8ac142599731c323ca71483a879.png)](https://yaroslavgrebnov.com/projects/workers/description)

**Workers** is a **lightweight Go library** for executing **multiple tasks concurrently**, with support for either a dynamic or fixed number of workers. Written by [ygrebnov](https://github.com/ygrebnov).

Designed to be **simple and easy to use**, it allows executing tasks concurrently with minimal setup. The library is suitable for a variety of use cases, from simple parallel processing to more complex workflows.

---

[![GoDoc](https://pkg.go.dev/badge/github.com/ygrebnov/workers)](https://pkg.go.dev/github.com/ygrebnov/workers)
[![Build Status](https://github.com/ygrebnov/workers/actions/workflows/build.yml/badge.svg)](https://github.com/ygrebnov/workers/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/ygrebnov/workers/graph/badge.svg?token=1TY5NH8IF6)](https://codecov.io/gh/ygrebnov/workers)
[![Go Report Card](https://goreportcard.com/badge/github.com/ygrebnov/workers)](https://goreportcard.com/report/github.com/ygrebnov/workers)

[User Guide](https://yaroslavgrebnov.com/projects/workers/description) | [Examples](https://yaroslavgrebnov.com/projects/workers/examples) | [Contributing](#contributing)

## Features

- **Flexible Worker Pools:** Execute tasks concurrently using either dynamic (scalable) or fixed-size worker pools.
- **Streaming Results and Errors:** Task results and errors are provided via channels.
- **Configurable Error Handling:** Option to stop immediately upon encountering an error or continue execution.
- **Robust Panic Recovery:** Safely handles panics within tasks, preventing crashes.
- **Delayed Task Execution:** Tasks can be accumulated first and executed later on-demand.

## Installation

Requires Go 1.22.3 or later:

```shell
go get github.com/ygrebnov/workers
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

	for i := 20; i >= 11; i-- {
		// Add ten tasks calculating the Fibonacci number for a given index.
		// A task is a function that takes a context and returns a string.
		err := w.AddTask(func(ctx context.Context) string {
			return fmt.Sprintf("Calculated Fibonacci for: %d, result: %d.", i, fibonacci(i))
		})
		if err != nil {
			log.Fatalf("failed to add task: %v", err)
		}
	}

	// Receive and print results or handle errors.
	for range 10 {
		select {
		case result := <-w.GetResults():
			fmt.Println(result)
		case err := <-w.GetErrors():
			fmt.Println("error executing task:", err)
		}
	}

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