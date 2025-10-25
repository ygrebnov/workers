package workers

import "errors"

const Namespace = "workers"

var (
	ErrInvalidState = errors.New(
		Namespace + ": cannot add a task for non-started workers with unbuffered tasks channel",
	)
	ErrInvalidTaskType = errors.New(Namespace + ": invalid task type")
	ErrTaskCancelled   = errors.New(Namespace + ": task execution cancelled")
	ErrTaskPanicked    = errors.New(Namespace + ": task execution panicked")
	ErrInvalidOption   = errors.New(Namespace + ": invalid option")
	ErrInvalidConfig   = errors.New(Namespace + ": invalid configuration")
)
