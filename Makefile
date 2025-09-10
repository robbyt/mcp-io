.PHONY: all
all: help

## help: Display this help message
.PHONY: help
help: Makefile
	@echo
	@echo " Choose a make command to run"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'
	@echo

## test: Run tests with race detection and coverage
.PHONY: test
test:
	go test -race -cover ./...

## test-verbose: Run tests with verbose output
.PHONY: test-verbose
test-verbose:
	go test -race -cover -v ./...

## bench: Run performance benchmarks
.PHONY: bench
bench:
	go test -run=^$$ -bench=. -benchmem ./...

## lint: Run golangci-lint code quality checks
.PHONY: lint
lint:
	golangci-lint run ./...

## lint-fix: Run golangci-lint with auto-fix for common issues
.PHONY: lint-fix
lint-fix:
	golangci-lint fmt
	golangci-lint run --fix ./...

## tidy: Clean up go modules
.PHONY: tidy
tidy:
	go mod tidy
	go mod verify

## fmt: Format Go source code
.PHONY: fmt
fmt:
	go fmt ./...

## vet: Run go vet static analysis
.PHONY: vet
vet:
	go vet ./...

## build: Build the library (check for compile errors)
.PHONY: build
build:
	go build ./...

## build-examples: Build all example applications
.PHONY: build-examples
build-examples: build-http-server build-cli-tool

## build-http-server: Build the HTTP server example
.PHONY: build-http-server
build-http-server:
	@mkdir -p bin/
	go build -o bin/http-server ./examples/http_server/

## build-cli-tool: Build the CLI tool example
.PHONY: build-cli-tool
build-cli-tool:
	@mkdir -p bin/
	go build -o bin/cli-tool ./examples/cli_tool/

## clean: Clean up build artifacts and caches
.PHONY: clean
clean:
	rm -rf bin/

## check: Run all checks (fmt, vet, lint, test)
.PHONY: check
check: fmt vet lint test