package pool

type fixed struct {
	available chan interface{}
	len       chan struct{}
	newFn     func() interface{}
}

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
