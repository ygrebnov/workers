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

bench: dir-coverage
	@go test ./tests/... -memprofile mem.prof -bench=. --run BenchmarkWorkers
	@go tool pprof -http :8080 mem.prof

test-pprof: dir-profiling
	@go test ./tests/... -memprofile $(PROFILING_PATH)mem.prof
	@go tool pprof -http :8080 $(PROFILING_PATH)mem.prof

.PHONY: clean dir-coverage dir-profiling lint test bench test-pprof