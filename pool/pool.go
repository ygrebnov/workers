package pool

type Pool interface {
	Get() interface{}
	Put(interface{})
}
