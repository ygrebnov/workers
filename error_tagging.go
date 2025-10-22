package workers

import (
	"errors"
	"fmt"
)

// TaskMetaError exposes correlation metadata for a task failure.
type TaskMetaError interface {
	error
	Unwrap() error
	TaskID() (any, bool)
	TaskIndex() (int, bool)
}

type taskTaggedError struct {
	err   error
	id    any
	index int
}

func newTaskTaggedError(err error, id any, index int) error {
	if err == nil {
		return nil
	}
	return &taskTaggedError{err: err, id: id, index: index}
}

func (e *taskTaggedError) Error() string { return e.err.Error() }
func (e *taskTaggedError) Unwrap() error { return e.err }

func (e *taskTaggedError) TaskID() (any, bool) {
	if e.id == nil {
		return nil, false
	}
	return e.id, true
}

func (e *taskTaggedError) TaskIndex() (int, bool) { return e.index, true }

func (e *taskTaggedError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "task(index=%d,id=%v): %+v", e.index, e.id, e.err)
			return
		}
		fallthrough
	case 's':
		_, _ = fmt.Fprint(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	}
}

// ExtractTaskID returns the task ID from err if present.
func ExtractTaskID(err error) (any, bool) {
	var tme TaskMetaError
	if errors.As(err, &tme) {
		return tme.TaskID()
	}
	return nil, false
}

// ExtractTaskIndex returns the task index from err if present.
func ExtractTaskIndex(err error) (int, bool) {
	var tme TaskMetaError
	if errors.As(err, &tme) {
		return tme.TaskIndex()
	}
	return 0, false
}
