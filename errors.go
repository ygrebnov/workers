package workers

import "errors"

var (
	ErrInvalidState    = errors.New("cannot add a task for non-started workers with unbuffered tasks channel")
	ErrInvalidTaskType = errors.New("invalid task type")
	ErrTaskCancelled   = errors.New("task execution cancelled")
	ErrTaskPanicked    = errors.New("task execution panicked")
)
