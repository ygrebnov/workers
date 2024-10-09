package pool

import "sync"

func NewDynamic(newFn func() interface{}) Pool {
	return &sync.Pool{New: newFn}
}
