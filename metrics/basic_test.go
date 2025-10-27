package metrics

import (
	"reflect"
	"runtime"
	"sync"
	"testing"
)

func TestBasicProvider_Counter_ReusedAndAccumulates(t *testing.T) {
	p := NewBasicProvider()

	c1 := p.Counter("tasks_enqueued")
	c2 := p.Counter("tasks_enqueued")

	if reflect.ValueOf(c1).Pointer() != reflect.ValueOf(c2).Pointer() {
		t.Fatalf("expected same counter instance for same name")
	}

	// Access concrete type to assert snapshot values.
	bc, ok := c1.(*BasicCounter)
	if !ok {
		t.Fatalf("expected *BasicCounter, got %T", c1)
	}

	c1.Add(3)
	c2.Add(2)
	if got := bc.Snapshot(); got != 5 {
		t.Fatalf("counter value = %d; want 5", got)
	}

	// Different name -> different instance
	cOther := p.Counter("other")
	if reflect.ValueOf(cOther).Pointer() == reflect.ValueOf(c1).Pointer() {
		t.Fatalf("expected different counter instance for different name")
	}
}

func TestBasicProvider_UpDownCounter_ReusedAndMoves(t *testing.T) {
	p := NewBasicProvider()
	u1 := p.UpDownCounter("inflight")
	u2 := p.UpDownCounter("inflight")

	if reflect.ValueOf(u1).Pointer() != reflect.ValueOf(u2).Pointer() {
		t.Fatalf("expected same updown instance for same name")
	}

	bu, ok := u1.(*BasicUpDownCounter)
	if !ok {
		t.Fatalf("expected *BasicUpDownCounter, got %T", u1)
	}

	u1.Add(+3)
	u2.Add(-1)
	u1.Add(+10)
	if got := bu.Snapshot(); got != 12 {
		t.Fatalf("updown value = %d; want 12", got)
	}
}

func TestBasicProvider_Histogram_RecordsStats(t *testing.T) {
	p := NewBasicProvider()
	h := p.Histogram("exec_seconds")

	bh, ok := h.(*BasicHistogram)
	if !ok {
		t.Fatalf("expected *BasicHistogram, got %T", h)
	}

	h.Record(0.1)
	h.Record(0.3)
	h.Record(0.2)
	s := bh.Snapshot()
	if s.Count != 3 {
		t.Fatalf("count = %d; want 3", s.Count)
	}
	if s.Min != 0.1 || s.Max != 0.3 {
		t.Fatalf("min/max = (%v,%v); want (0.1,0.3)", s.Min, s.Max)
	}
	if s.Sum < 0.59 || s.Sum > 0.61 {
		t.Fatalf("sum = %v; want ~0.6", s.Sum)
	}
	if s.Mean < 0.19 || s.Mean > 0.21 {
		t.Fatalf("mean = %v; want ~0.2", s.Mean)
	}
}

func TestBasicProvider_Concurrent_GetSameInstrument(t *testing.T) {
	p := NewBasicProvider()
	n := 50
	ptrs := make([]uintptr, n)
	wg := sync.WaitGroup{}
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			c := p.Counter("shared")
			ptrs[idx] = reflect.ValueOf(c).Pointer()
		}(i)
	}
	wg.Wait()
	first := ptrs[0]
	for i := 1; i < n; i++ {
		if ptrs[i] != first {
			t.Fatalf("expected same pointer for all retrieved counters; mismatch at %d", i)
		}
	}
}

func TestBasicProvider_Concurrent_CounterAdd(t *testing.T) {
	p := NewBasicProvider()
	c := p.Counter("hits")
	bc := c.(*BasicCounter)

	workers := runtime.NumCPU() * 2
	iters := 1000
	wg := sync.WaitGroup{}
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				c.Add(1)
			}
		}()
	}
	wg.Wait()
	expected := int64(workers * iters)
	if got := bc.Snapshot(); got != expected {
		t.Fatalf("counter = %d; want %d", got, expected)
	}
}

func TestBasicProvider_Concurrent_UpDownAdd(t *testing.T) {
	p := NewBasicProvider()
	u := p.UpDownCounter("inflight")
	bu := u.(*BasicUpDownCounter)

	workers := runtime.NumCPU() * 2
	iters := 1000
	wg := sync.WaitGroup{}
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				if (i+id)%2 == 0 {
					u.Add(+1)
				} else {
					u.Add(-1)
				}
			}
		}(w)
	}
	wg.Wait()
	// Even distribution; value may not be exactly zero depending on parity, compute expected
	expected := int64(0)
	// Each worker does iters ops; across workers, half +1 and half -1 on average
	if got := bu.Snapshot(); got != expected {
		// allow small drift only if test logic changes; for now enforce exact zero
		t.Fatalf("updown = %d; want %d", got, expected)
	}
}

func TestBasicProvider_Concurrent_HistogramRecord(t *testing.T) {
	p := NewBasicProvider()
	h := p.Histogram("latency")
	bh := h.(*BasicHistogram)

	workers := runtime.NumCPU() * 2
	iters := 500
	wg := sync.WaitGroup{}
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(base int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				// record a few bounded values
				v := float64((base%10)+i%10) / 100.0
				h.Record(v)
			}
		}(w)
	}
	wg.Wait()
	s := bh.Snapshot()
	expectedCount := int64(workers * iters)
	if s.Count != expectedCount {
		t.Fatalf("hist count = %d; want %d", s.Count, expectedCount)
	}
	if s.Min < 0.0 || s.Min > 0.09 || s.Max < 0.0 || s.Max > 0.19 {
		t.Fatalf("min/max out of expected range: (%v,%v)", s.Min, s.Max)
	}
}
