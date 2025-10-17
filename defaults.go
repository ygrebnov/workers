package workers

// defaultConfig centralizes default values for Config.
// These defaults are applied by both New (when cfg is nil) and NewOptions (options builder base).
func defaultConfig() Config {
	return Config{
		MaxWorkers:                  0,     // dynamic pool
		StartImmediately:            false, // explicit Start by default
		StopOnError:                 false,
		TasksBufferSize:             0,
		ResultsBufferSize:           1024,
		ErrorsBufferSize:            1024,
		StopOnErrorErrorsBufferSize: 100,
	}
}

// validateConfig performs lightweight invariants checks.
// It returns nil for all currently valid states; reserved for future validation expansions.
func validateConfig(cfg *Config) error {
	// MaxWorkers == 0 -> dynamic pool; >0 -> fixed-size pool.
	// All buffer sizes are uint; zero is a valid choice (unbuffered) except we provide non-zero defaults above.
	// No hard validation required at the moment.
	return nil
}
