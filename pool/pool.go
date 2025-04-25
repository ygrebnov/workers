package pool

// Pool is an interface that defines methods on a pool of workers.
type Pool interface {
	// Get returns a worker from the pool.
	Get() interface{}

	// Put returns a worker back to the pool.
	Put(interface{})
}
