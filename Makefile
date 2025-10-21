ROOT_PATH := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))
COVERAGE_PATH := $(ROOT_PATH).coverage/
PROFILING_PATH := $(ROOT_PATH).profiling/

include $(CURDIR)/tools/tools.mk

clean:
	@rm -rf $(BUILD_PATH)

dir-coverage:
	@mkdir -p $(COVERAGE_PATH)

dir-profiling:
	@mkdir -p $(PROFILING_PATH)

lint: install-golangci-lint
	$(GOLANGCI_LINT) run

test: dir-coverage
	@go test -v -coverpkg=./... ./... -coverprofile $(COVERAGE_PATH)coverage.txt
	@go tool cover -func=$(COVERAGE_PATH)coverage.txt -o $(COVERAGE_PATH)functions.txt
	@go tool cover -html=$(COVERAGE_PATH)coverage.txt -o $(COVERAGE_PATH)coverage.html

# Run only the FIFO vs Pools benchmarks with memory stats and save output
bench: dir-coverage
	@echo "Running FIFO vs Pools benchmarks..."
	@go test -run ^$$ -bench ^BenchmarkFIFO_vs_Pools -benchmem ./tests | tee $(COVERAGE_PATH)bench.txt

# Save timestamped benchmark results for historical comparisons
bench-save: dir-coverage
	@ts=$$(date +%Y%m%d_%H%M%S); \
	echo "Saving benchmarks to $(COVERAGE_PATH)bench_$${ts}.txt"; \
	go test -run ^$$ -bench ^BenchmarkFIFO_vs_Pools -benchmem ./tests > $(COVERAGE_PATH)bench_$${ts}.txt

# Memory profiling for tests (not benchmarks)
test-pprof: dir-profiling
	@go test ./tests/... -memprofile $(PROFILING_PATH)mem.prof
	@go tool pprof -http :8080 $(PROFILING_PATH)mem.prof

.PHONY: clean dir-coverage dir-profiling lint test bench bench-save test-pprof
