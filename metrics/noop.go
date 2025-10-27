package metrics

// NoopProvider returns no-op instruments. Useful as the default provider.
// All methods are safe for concurrent use and perform no work.
type NoopProvider struct{}

// NewNoopProvider constructs a Provider that discards all metrics.
func NewNoopProvider() NoopProvider { return NoopProvider{} }

func (NoopProvider) Counter(_ string, _ ...InstrumentOption) Counter {
	return noopCounter{}
}

func (NoopProvider) UpDownCounter(_ string, _ ...InstrumentOption) UpDownCounter {
	return noopUpDownCounter{}
}

func (NoopProvider) Histogram(_ string, _ ...InstrumentOption) Histogram {
	return noopHistogram{}
}

type noopCounter struct{}

func (noopCounter) Add(_ int64) {}

type noopUpDownCounter struct{}

func (noopUpDownCounter) Add(_ int64) {}

type noopHistogram struct{}

func (noopHistogram) Record(_ float64) {}
