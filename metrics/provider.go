package metrics

// Provider constructs instruments used to record metrics.
// Implementations must be safe for concurrent use.
//
// Keep this interface minimal and stable. If you need new capabilities later,
// introduce separate optional interfaces rather than expanding this surface.
type Provider interface {
	Counter(name string, opts ...InstrumentOption) Counter
	UpDownCounter(name string, opts ...InstrumentOption) UpDownCounter
	Histogram(name string, opts ...InstrumentOption) Histogram
}

// Counter records monotonic counts.
// Methods must be safe for concurrent use.
type Counter interface {
	Add(n int64)
}

// UpDownCounter records values that can move up or down (e.g., current in-flight).
// Methods must be safe for concurrent use.
type UpDownCounter interface {
	Add(n int64)
}

// Histogram records distribution of float64 measurements (e.g., durations in seconds).
// Methods must be safe for concurrent use.
type Histogram interface {
	Record(v float64)
}

// InstrumentConfig carries optional instrument metadata. It's advisory only.
type InstrumentConfig struct {
	Description string
	Unit        string
	// Attributes are static key-value pairs associated with the instrument itself.
	// Keep cardinality bounded. Implementations may ignore attributes.
	Attributes map[string]string
}

// InstrumentOption mutates InstrumentConfig.
type InstrumentOption func(*InstrumentConfig)

// WithDescription sets an advisory description for the instrument.
func WithDescription(desc string) InstrumentOption {
	return func(c *InstrumentConfig) { c.Description = desc }
}

// WithUnit sets an advisory unit for the instrument (e.g., "1", "seconds").
func WithUnit(unit string) InstrumentOption {
	return func(c *InstrumentConfig) { c.Unit = unit }
}

// WithAttributes attaches static attributes to the instrument (bounded cardinality only).
func WithAttributes(attrs map[string]string) InstrumentOption {
	return func(c *InstrumentConfig) {
		if len(attrs) == 0 {
			return
		}
		// copy to avoid external mutation
		if c.Attributes == nil {
			c.Attributes = make(map[string]string, len(attrs))
		}
		for k, v := range attrs {
			c.Attributes[k] = v
		}
	}
}
