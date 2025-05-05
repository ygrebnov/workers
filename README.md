[![Build Status](https://github.com/ygrebnov/workers/actions/workflows/build.yml/badge.svg)](https://github.com/ygrebnov/workers/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/ygrebnov/workers/graph/badge.svg?token=1TY5NH8IF6)](https://codecov.io/gh/ygrebnov/workers)

Package workers
===============

Package workers provides functionality of asynchronous workers executing tasks.

Features:

- execution with a variable (dynamic) number of workers. sync.Pool based implementation, suitable for most cases,
- execution with a number of workers limited to a fixed number. Preferred for execution of tasks demanding significant amount of memory allocation on start,
- supports execution of tasks with different signatures:
  - func(context.Context) (Result, error)
  - func(context.Context) (Result)
  - func(context.Context) (error)
- tasks execution results streaming via channels,
- supports delayed tasks execution start.

Installation
____________

```shell
go get github.com/ygrebnov/workers
```

License
-------

Distributed under MIT License, please see license file within the code for more details.