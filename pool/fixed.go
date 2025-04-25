package pool

type fixed struct {
	available chan interface{}
	len       chan struct{}
	newFn     func() interface{}
}

// NewFixed creates a new fixed-size pool of workers with the given capacity and a function to create new workers.
// The pool size is limited to the specified capacity, and if all workers are in use,
// it blocks until one is returned to the pool and becomes available.
func NewFixed(capacity uint, newFn func() interface{}) Pool {
	return &fixed{
		available: make(chan interface{}, capacity),
		len:       make(chan struct{}, capacity),
		newFn:     newFn,
	}
}

func (p *fixed) Get() interface{} {
	select {
	case el := <-p.available:
		return el

	case p.len <- struct{}{}:
		return p.newFn()
	}
}

func (p *fixed) Put(el interface{}) {
	p.available <- el
}
