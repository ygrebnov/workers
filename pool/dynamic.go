package pool

import "sync"

// NewDynamic is a dynamic-size pool of workers. It is a wrapper around sync.Pool.
func NewDynamic(newFn func() interface{}) Pool {
	return &sync.Pool{New: newFn}
}
