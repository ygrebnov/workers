ROOT_PATH := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))
COVERAGE_PATH := $(ROOT_PATH).coverage/
PROFILING_PATH := $(ROOT_PATH).profiling/
BENCH_PATH := $(ROOT_PATH).bench/

include $(CURDIR)/tools/tools.mk

# Path to benchstat installed by tools.mk
BENCHSTAT := $(CURDIR)/tools/bin/benchstat

clean:
	@go clean -testcache
	@rm -rf $(COVERAGE_PATH)

dir-coverage:
	@mkdir -p $(COVERAGE_PATH)

dir-profiling:
	@mkdir -p $(PROFILING_PATH)

dir-bench:
	@mkdir -p $(BENCH_PATH)

lint: install-golangci-lint
	$(GOLANGCI_LINT) run

test: dir-coverage
	@go test -v -coverpkg=./... ./... -coverprofile $(COVERAGE_PATH)coverage.txt
	@go tool cover -func=$(COVERAGE_PATH)coverage.txt -o $(COVERAGE_PATH)functions.txt
	@go tool cover -html=$(COVERAGE_PATH)coverage.txt -o $(COVERAGE_PATH)coverage.html

test-race:
	@go test ./... -race -count=1 -timeout=600s

# Run only the FIFO vs Pools benchmarks with memory stats and save output
bench: dir-bench
	@echo "Running FIFO vs Pools benchmarks..."
	@go test -run ^$$ -bench ^BenchmarkFIFO_vs_Pools -benchmem ./tests | tee $(BENCH_PATH)bench.txt

# Save timestamped benchmark results for historical comparisons
bench-save: dir-bench
	@ts=$$(date +%Y%m%d_%H%M%S); \
	echo "Saving benchmarks to $(BENCH_PATH)bench_$${ts}.txt"; \
	go test -run ^$$ -bench ^BenchmarkFIFO_vs_Pools -benchmem ./tests > $(BENCH_PATH)bench_$${ts}.txt

# Compare two benchmark result files using benchstat
# Usage: make bench-compare OLD=.bench/bench_old.txt NEW=.bench/bench_new.txt
bench-compare: $(BENCHSTAT)
	@if [ -z "$(OLD)" ] || [ -z "$(NEW)" ]; then \
		echo "Usage: make bench-compare OLD=.bench/old.txt NEW=.bench/new.txt"; \
		exit 1; \
	fi
	@$(BENCHSTAT) $(OLD) $(NEW)

# Memory profiling for tests (not benchmarks)
test-pprof: dir-profiling
	@go test ./tests/... -memprofile $(PROFILING_PATH)mem.prof
	@go tool pprof -http :8080 $(PROFILING_PATH)mem.prof

.PHONY: clean dir-coverage dir-profiling dir-bench lint test test-race bench bench-save bench-compare test-pprof
