package metrics

import (
	"math"
	"sync"
	"sync/atomic"
)

// BasicProvider is a simple in-memory implementation of Provider.
// It is concurrency-safe and suitable for tests, examples, and lightweight apps.
// Instruments are created on demand by name and reused for the same name.
// Instrument options are currently advisory and stored for potential introspection.
type BasicProvider struct {
	mu         sync.RWMutex
	counters   map[string]*BasicCounter
	updowns    map[string]*BasicUpDownCounter
	histograms map[string]*BasicHistogram
	meta       map[string]InstrumentConfig // optional stored metadata per name
}

// NewBasicProvider constructs a new BasicProvider.
func NewBasicProvider() *BasicProvider {
	return &BasicProvider{
		counters:   make(map[string]*BasicCounter),
		updowns:    make(map[string]*BasicUpDownCounter),
		histograms: make(map[string]*BasicHistogram),
		meta:       make(map[string]InstrumentConfig),
	}
}

// applyOptions builds InstrumentConfig from options.
func applyOptions(opts []InstrumentOption) InstrumentConfig {
	var cfg InstrumentConfig
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	return cfg
}

// Counter returns a monotonic counter instrument for the given name (created once).
func (p *BasicProvider) Counter(name string, opts ...InstrumentOption) Counter {
	p.mu.RLock()
	c, ok := p.counters[name]
	if ok {
		p.mu.RUnlock()
		return c
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	// re-check after acquiring write lock
	if c, ok = p.counters[name]; ok {
		return c
	}
	cfg := applyOptions(opts)
	p.meta[name] = cfg
	c = &BasicCounter{}
	p.counters[name] = c
	return c
}

// UpDownCounter returns an up/down counter instrument for the given name (created once).
func (p *BasicProvider) UpDownCounter(name string, opts ...InstrumentOption) UpDownCounter {
	p.mu.RLock()
	u, ok := p.updowns[name]
	if ok {
		p.mu.RUnlock()
		return u
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if u, ok = p.updowns[name]; ok {
		return u
	}
	cfg := applyOptions(opts)
	p.meta[name] = cfg
	u = &BasicUpDownCounter{}
	p.updowns[name] = u
	return u
}

// Histogram returns a histogram instrument for the given name (created once).
func (p *BasicProvider) Histogram(name string, opts ...InstrumentOption) Histogram {
	p.mu.RLock()
	h, ok := p.histograms[name]
	if ok {
		p.mu.RUnlock()
		return h
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if h, ok = p.histograms[name]; ok {
		return h
	}
	cfg := applyOptions(opts)
	p.meta[name] = cfg
	h = &BasicHistogram{min: math.Inf(1), max: math.Inf(-1)}
	p.histograms[name] = h
	return h
}

// BasicCounter is a thread-safe monotonic counter.
type BasicCounter struct {
	val atomic.Int64
}

// Add increments the counter by n (n may be negative but it's not recommended for monotonic counters).
func (c *BasicCounter) Add(n int64) { c.val.Add(n) }

// Snapshot returns the current value.
func (c *BasicCounter) Snapshot() int64 { return c.val.Load() }

// BasicUpDownCounter is a thread-safe up/down counter.
type BasicUpDownCounter struct {
	val atomic.Int64
}

// Add adds n (positive or negative) to the current value.
func (u *BasicUpDownCounter) Add(n int64) { u.val.Add(n) }

// Snapshot returns the current value.
func (u *BasicUpDownCounter) Snapshot() int64 { return u.val.Load() }

// BasicHistogram is a thread-safe histogram that tracks count, sum, min, and max.
// It does not maintain buckets; it's intended as a lightweight, general-purpose aggregator.
type BasicHistogram struct {
	mu    sync.Mutex
	count int64
	sum   float64
	min   float64
	max   float64
}

// Record adds a measurement to the histogram.
func (h *BasicHistogram) Record(v float64) {
	h.mu.Lock()
	if h.count == 0 {
		// initialize min/max on first record
		h.min, h.max = v, v
	} else {
		if v < h.min {
			h.min = v
		}
		if v > h.max {
			h.max = v
		}
	}
	h.count++
	h.sum += v
	h.mu.Unlock()
}

// HistSnapshot is an immutable snapshot of a BasicHistogram.
type HistSnapshot struct {
	Count int64
	Sum   float64
	Min   float64
	Max   float64
	Mean  float64
}

// Snapshot returns a copy of the histogram state at the time of call.
func (h *BasicHistogram) Snapshot() HistSnapshot {
	h.mu.Lock()
	count := h.count
	sum := h.sum
	min := h.min
	max := h.max
	h.mu.Unlock()
	mean := 0.0
	if count > 0 {
		mean = sum / float64(count)
	}
	return HistSnapshot{Count: count, Sum: sum, Min: min, Max: max, Mean: mean}
}
