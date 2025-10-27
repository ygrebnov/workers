package workers

import "errors"

const Namespace = "workers"

var (
	ErrInvalidState = errors.New(
		Namespace + ": cannot add a task for non-started workers with unbuffered tasks channel",
	)
	ErrExclusiveIntakeChannel = errors.New(
		Namespace + ": cannot add a task when a user-supplied intake channel is configured",
	)
	ErrTaskCancelled = errors.New(Namespace + ": task execution cancelled")
	ErrTaskPanicked  = errors.New(Namespace + ": task execution panicked")
	ErrInvalidConfig = errors.New(Namespace + ": invalid configuration")
)
