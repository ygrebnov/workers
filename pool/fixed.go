package pool

type fixed struct {
	available chan interface{}
	all       chan interface{}
	buf       chan interface{}
	newFn     func() interface{}
}

func NewFixed(capacity uint, newFn func() interface{}) Pool {
	return &fixed{
		available: make(chan interface{}, capacity),
		all:       make(chan interface{}, capacity),
		buf:       make(chan interface{}, 1024),
		newFn:     newFn,
	}
}

func (p *fixed) Get() interface{} {
	select {
	case el := <-p.available:
		return el

	case el := <-p.buf:
		return el

	default:
		var el interface{}

		if len(p.all) < cap(p.all) {
			el = p.newFn()
		} else {
			el = <-p.all
		}

		select {
		case p.all <- el:
		case p.buf <- el:
		default:
		}
		return el
	}
}

func (p *fixed) Put(el interface{}) {
	select {
	case p.available <- el:
	case p.all <- el:
	case p.buf <- el:
	default:
	}
}
