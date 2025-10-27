package metrics

// NoopProvider returns no-op instruments. Useful as the default provider.
// All methods are safe for concurrent use and perform no work.
type NoopProvider struct{}

// NewNoopProvider constructs a Provider that discards all metrics.
func NewNoopProvider() NoopProvider { return NoopProvider{} }

func (NoopProvider) Counter(name string, opts ...InstrumentOption) Counter {
	return noopCounter{}
}

func (NoopProvider) UpDownCounter(name string, opts ...InstrumentOption) UpDownCounter {
	return noopUpDownCounter{}
}

func (NoopProvider) Histogram(name string, opts ...InstrumentOption) Histogram {
	return noopHistogram{}
}

type noopCounter struct{}

func (noopCounter) Add(n int64) {}

type noopUpDownCounter struct{}

func (noopUpDownCounter) Add(n int64) {}

type noopHistogram struct{}

func (noopHistogram) Record(v float64) {}
