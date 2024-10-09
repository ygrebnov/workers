package pool

type fixed struct {
	available chan interface{}
	all       chan interface{}
	newFn     func() interface{}
}

func NewFixed(capacity uint, newFn func() interface{}) Pool {
	return &fixed{
		available: make(chan interface{}, capacity),
		all:       make(chan interface{}, capacity),
		newFn:     newFn,
	}
}

func (p *fixed) Get() interface{} {
	select {
	case el := <-p.available:
		return el

	default:
		if len(p.all) < cap(p.all) {
			el := p.newFn()
			p.all <- el
			return el
		}

		el := <-p.all
		p.all <- el
		return el
	}
}

func (p *fixed) Put(el interface{}) {
	p.available <- el
}
