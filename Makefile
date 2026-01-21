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

## test: Run all tests including isolated modules
.PHONY: test
test: test-core test-adk

## test-core: Run tests for core library (unit + integration) with coverage
.PHONY: test-core
test-core:
	go test -race -cover -tags $(ALL_BUILD_TAGS) ./...

## test-adk: Run tests for ADK integration example
.PHONY: test-adk
test-adk: workspace-init
	go -C examples/adk_integration test -race -cover ./...

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

## workspace-init: Initialize Go workspace for multi-module development
.PHONY: workspace-init
workspace-init: go.work

go.work:
	go work init . ./examples/adk_integration

## tidy: Clean up go modules (root only, keeps modules independent)
.PHONY: tidy
tidy:
	go mod tidy
	go mod verify

## tidy-workspace: Tidy both root and workspace modules independently
.PHONY: tidy-workspace
tidy-workspace: workspace-init
	go mod tidy
	go mod verify
	go -C examples/adk_integration mod tidy

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
build-examples: build-cli-simple build-cli-prompt build-cli-resource build-mixed-resources build-cli-elicitation build-simple-dungeon-master build-cli-agent build-cli-completion build-http-multistep build-schema-flexibility build-adk-integration

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

## build-cli-agent: Build the CLI agent example
.PHONY: build-cli-agent
build-cli-agent:
	@mkdir -p bin/
	go build -o bin/cli-agent ./examples/cli_agent/

## build-cli-completion: Build the CLI completion example
.PHONY: build-cli-completion
build-cli-completion:
	@mkdir -p bin/
	go build -o bin/cli-completion ./examples/cli_completion/

## build-http-multistep: Build the HTTP multistep example
.PHONY: build-http-multistep
build-http-multistep:
	@mkdir -p bin/
	go build -o bin/http-multistep ./examples/http_multistep/

## build-schema-flexibility: Build the schema flexibility example
.PHONY: build-schema-flexibility
build-schema-flexibility:
	@mkdir -p bin/
	go build -o bin/schema-flexibility ./examples/schema_flexibility/

## build-adk-integration: Build the ADK integration example
.PHONY: build-adk-integration
build-adk-integration: workspace-init
	@mkdir -p bin/
	go -C examples/adk_integration build -o ../../bin/adk-integration .

## clean: Clean up build artifacts and caches
.PHONY: clean
clean:
	rm -rf bin/

## check: Run all checks (fmt, vet, lint, test)
.PHONY: check
check: fmt vet lint test
