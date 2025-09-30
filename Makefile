ALL_BUILD_TAGS := "integration"

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

## test: Run all tests (unit + integration) with coverage
.PHONY: test
test:
	go test -race -cover -tags $(ALL_BUILD_TAGS) ./...

## test-short: Run tests in short mode (fast, no coverage)
.PHONY: test-short
test-short:
	go test -race -short ./...

## bench: Run performance benchmarks
.PHONY: bench
bench:
	go test -run=^$$ -bench=. -benchmem ./...

## lint: Run golangci-lint code quality checks
.PHONY: lint
lint:
	golangci-lint run --build-tags $(ALL_BUILD_TAGS) ./...

## lint-fix: Run golangci-lint with auto-fix for common issues
.PHONY: lint-fix
lint-fix:
	golangci-lint fmt
	golangci-lint run --build-tags $(ALL_BUILD_TAGS) --fix ./...

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
build-examples: build-http-server build-cli-tool build-cli-prompt build-cli-resource build-cli-combined build-cli-elicitation

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

## build-cli-prompt: Build the CLI prompt example
.PHONY: build-cli-prompt
build-cli-prompt:
	@mkdir -p bin/
	go build -o bin/cli-prompt ./examples/cli_prompt/

## build-cli-resource: Build the CLI resource example
.PHONY: build-cli-resource
build-cli-resource:
	@mkdir -p bin/
	go build -o bin/cli-resource ./examples/cli_resource/

## build-cli-combined: Build the CLI combined example
.PHONY: build-cli-combined
build-cli-combined:
	@mkdir -p bin/
	go build -o bin/cli-combined ./examples/cli_combined/

## build-cli-elicitation: Build the CLI elicitation example
.PHONY: build-cli-elicitation
build-cli-elicitation:
	@mkdir -p bin/
	go build -o bin/cli-elicitation ./examples/cli_elicitation/

## clean: Clean up build artifacts and caches
.PHONY: clean
clean:
	rm -rf bin/

## check: Run all checks (fmt, vet, lint, test)
.PHONY: check
check: fmt vet lint test
