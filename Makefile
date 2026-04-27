MODULES := . snowflake uuidv4 uuidv7 ulid

.PHONY: test test-verbose test-race bench fmt vet tidy help

help:
	@echo "Available commands:"
	@echo "  make test         - Run all tests across all modules"
	@echo "  make test-verbose - Run tests with verbose output"
	@echo "  make test-race    - Run tests with race detector"
	@echo "  make bench        - Run benchmarks"
	@echo "  make fmt          - Format all code"
	@echo "  make vet          - Run go vet on all modules"
	@echo "  make tidy         - Run go mod tidy on all modules"

test:
	@for m in $(MODULES); do echo "=== $$m ===" && cd $$m && go test ./... && cd ..; done

test-verbose:
	@for m in $(MODULES); do echo "=== $$m ===" && cd $$m && go test -v ./... && cd ..; done

test-race:
	@for m in $(MODULES); do echo "=== $$m ===" && cd $$m && go test -race ./... && cd ..; done

bench:
	@for m in $(MODULES); do echo "=== $$m ===" && cd $$m && go test -bench=. -benchmem ./... && cd ..; done

fmt:
	@for m in $(MODULES); do cd $$m && gofmt -s -w . && cd ..; done

vet:
	@for m in $(MODULES); do echo "=== $$m ===" && cd $$m && go vet ./... && cd ..; done

tidy:
	@for m in $(MODULES); do echo "=== $$m ===" && cd $$m && go mod tidy && cd ..; done
