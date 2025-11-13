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

## test: Run all tests for core library (unit + integration) with coverage
.PHONY: test
test:
	go test -race -cover -tags $(ALL_BUILD_TAGS) ./...

## bench: Run performance benchmarks
.PHONY: bench
bench:
	go test -run=^$$ -bench=. -benchmem ./...

## test-adk: Run tests for ADK integration example
.PHONY: test-adk
test-adk:
	go -C examples/adk_integration test -race -cover ./...

## test-all: Run all tests including isolated modules
.PHONY: test-all
test-all: test test-adk

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
build-examples: build-cli-simple build-cli-prompt build-cli-resource build-mixed-resources build-cli-elicitation build-simple-dungeon-master

## build-cli-simple: Build the simple example (supports both stdio and HTTP)
.PHONY: build-cli-simple
build-cli-simple:
	@mkdir -p bin/
	go build -o bin/cli-simple ./examples/cli_simple/

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

## build-mixed-resources: Build the mixed resources example
.PHONY: build-mixed-resources
build-mixed-resources:
	@mkdir -p bin/
	go build -o bin/mixed-resources ./examples/mixed_resources/

## build-cli-elicitation: Build the CLI elicitation example
.PHONY: build-cli-elicitation
build-cli-elicitation:
	@mkdir -p bin/
	go build -o bin/cli-elicitation ./examples/cli_elicitation/

## build-simple-dungeon-master: Build the simple dungeon master example
.PHONY: build-simple-dungeon-master
build-simple-dungeon-master:
	@mkdir -p bin/
	go build -o bin/simple-dungeon-master ./examples/simple_dungeon_master/

## clean: Clean up build artifacts and caches
.PHONY: clean
clean:
	rm -rf bin/

## check: Run all checks (fmt, vet, lint, test)
.PHONY: check
check: fmt vet lint test
