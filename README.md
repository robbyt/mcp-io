# mcp-io

[![Go Reference](https://pkg.go.dev/badge/github.com/robbyt/go-mcpio.svg)](https://pkg.go.dev/github.com/robbyt/go-mcpio)
[![Go Report Card](https://goreportcard.com/badge/github.com/robbyt/go-mcpio)](https://goreportcard.com/report/github.com/robbyt/go-mcpio)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A high-level Go library that wraps the official [Model Context Protocol (MCP) SDK](https://github.com/modelcontextprotocol/go-sdk) to provide a functional options API with error handling for creating MCP servers.

## Overview

`mcp-io` provides a simple wrapper around the MCP SDK that:
- **Prevents panics** - Returns errors instead of panicking on invalid configuration
- **Functional options** - Clean, composable API for tool registration
- **Type safety** - Compile-time validation with automatic JSON schema generation
- **Error handling** - Structured error types and graceful failure handling

## Key Features

- **Graceful Error Handling**: Configuration errors return meaningful error messages instead of panicking
- **Functional Options**: Clean, composable API using the options pattern
- **Type-Safe Tools**: Define tools with Go types and automatic schema generation
- **Multiple Transports**: HTTP, SSE, and stdio support through a single handler
- **MCP SDK Compatible**: Built on the official MCP SDK with error safety added

## Installation

```bash
go get github.com/robbyt/go-mcpio
```

## Quick Start

Here's a simple example showing the basic API:

```go
package main

import (
	"context"
	"log"
	"net/http/httptest"
	"strings"

	mcpio "github.com/robbyt/go-mcpio"
)

// Define your input and output types
type TextInput struct {
	Text string `json:"text" jsonschema:"Text to transform"`
}

type TextOutput struct {
	Result string `json:"result" jsonschema:"Transformed text"`
}

// Tool function
func toUpper(ctx context.Context, input TextInput) (TextOutput, error) {
	return TextOutput{Result: strings.ToUpper(input.Text)}, nil
}

func main() {
	// Create an MCP handler with functional options
	handler, err := mcpio.New(
		mcpio.WithName("example-server"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
	)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	// Create a test server to demonstrate usage
	server := httptest.NewServer(handler)
	defer server.Close()

	log.Printf("MCP server created with tools: to_upper")
	log.Printf("Server created and ready to handle requests")
}
```

## Testing Your Server

You can test your MCP server using the [MCP CLI tools](https://github.com/f/mcptools):

```bash
# Install the CLI tool
go install github.com/f/mcptools/cmd/mcptools@latest

# Test your server (assuming it runs via stdio)
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | your-server-binary | jq

# Or use the MCP CLI directly with mcptools
mcptools tools your-server-binary
mcptools call tool_name --params '{"param":"value"}' your-server-binary
```

## Core Concepts

### Functional Options API

The library uses a functional options pattern for clean, composable configuration:

```go
handler, err := mcpio.New(
    mcpio.WithName("my-server"),
    mcpio.WithVersion("1.0.0"),
    mcpio.WithTool("tool1", "Description", toolFunc1),
    mcpio.WithTool("tool2", "Description", toolFunc2),
)
// Errors are returned, not panicked
if err != nil {
    log.Fatalf("Configuration error: %v", err)
}
```

### Transport Options

A single handler supports multiple transport types:

```go
// HTTP transport - standard web server
http.Handle("/mcp", handler)
log.Println("HTTP server listening on :8080/mcp")
http.ListenAndServe(":8080", nil)

// SSE transport - for browser clients
http.Handle("/mcp-sse", http.HandlerFunc(handler.ServeSSE))
log.Println("SSE server listening on :8080/mcp-sse")
http.ListenAndServe(":8080", nil)

// Stdio transport - for CLI tools
if err := handler.ServeStdio(os.Stdin, os.Stdout); err != nil {
    log.Fatal(err)
}
```

### Type-Safe Tools

Define tools using Go types with automatic schema generation. Use `jsonschema` struct tags to provide descriptions:

```go
type MyInput struct {
    Name string `json:"name" jsonschema:"User's full name"`
    Age  int    `json:"age"  jsonschema:"User's age in years"`
}
```

```go
type CalculateInput struct {
    Operation string  `json:"operation" jsonschema:"Arithmetic operation to perform"`
    A         float64 `json:"a" jsonschema:"First number"`
    B         float64 `json:"b" jsonschema:"Second number"`
}

type CalculateOutput struct {
    Result float64 `json:"result" jsonschema:"Calculation result"`
}

func calculate(ctx context.Context, input CalculateInput) (CalculateOutput, error) {
    var result float64
    switch input.Operation {
    case "add":
        result = input.A + input.B
    case "subtract":
        result = input.A - input.B
    case "multiply":
        result = input.A * input.B
    case "divide":
        if input.B == 0 {
            return CalculateOutput{}, mcpio.NewToolError("division by zero")
        }
        result = input.A / input.B
    default:
        return CalculateOutput{}, mcpio.ValidationError("unsupported operation: " + input.Operation)
    }
    return CalculateOutput{Result: result}, nil
}

// Add the tool with error handling
handler, err := mcpio.New(
    mcpio.WithName("calculator"),
    mcpio.WithTool("calculate", "Perform arithmetic operations", calculate),
)
if err != nil {
    log.Fatalf("Failed to register tool: %v", err)
}
```

## Advanced Features

### Raw JSON Tools

For dynamic schemas or JSON-to-JSON transformations:

```go
// Raw function that works with JSON bytes
processJSON := func(ctx context.Context, input []byte) ([]byte, error) {
    // Custom JSON processing logic here
    return []byte(`{"processed": true, "input_length": ` + fmt.Sprintf("%d", len(input)) + `}`), nil
}

// Define input schema for the raw tool
inputSchema := mcpio.CreateObjectSchema(
    "Raw processing input",
    map[string]string{
        "data": "Raw data to process",
    },
    []string{"data"},
)

handler, err := mcpio.New(
    mcpio.WithName("raw-processor"),
    mcpio.WithRawTool("process_raw", "Process raw JSON data", inputSchema, processJSON),
)
if err != nil {
    log.Fatalf("Failed to create raw tool: %v", err)
}
```

### Script Integration

For dynamic script-based tools:

```go
// Implement the ScriptEvaluator interface
type MyScriptEvaluator struct {
    // Your script engine implementation
}

func (e *MyScriptEvaluator) Execute(ctx context.Context, input []byte) ([]byte, error) {
    // Execute script and return result
    return []byte(`{"result": "script executed", "input_received": true}`), nil
}

func (e *MyScriptEvaluator) GetTimeout() time.Duration {
    return 5 * time.Second
}

// Register script tool
handler, err := mcpio.New(
    mcpio.WithName("script-server"),
    mcpio.WithScriptTool("lua_double", "Double the input using Lua", evaluator),
)
if err != nil {
    log.Fatalf("Failed to register script tool: %v", err)
}
```

## Error Handling

The library provides structured error handling instead of panics:

```go
// Configuration errors are returned, not panicked
handler, err := mcpio.New(
    mcpio.WithName(""), // Invalid: empty name
)
if err != nil {
    // Check for specific error types
    if errors.Is(err, mcpio.ErrEmptyName) {
        log.Printf("Server name cannot be empty")
        return
    }
    log.Printf("Configuration error: %v", err)
    return
}

// Tool errors are structured
func riskyTool(ctx context.Context, input MyInput) (MyOutput, error) {
    if input.Value < 0 {
        return MyOutput{}, mcpio.ValidationError("value must be positive")
    }
    
    result, err := doSomething(input)
    if err != nil {
        return MyOutput{}, mcpio.ProcessingError("processing failed: " + err.Error())
    }
    
    return MyOutput{Result: result}, nil
}
```

## Schema Generation

The library uses the same JSON schema generation as the MCP SDK:

```go
type MyInput struct {
    Name string `json:"name" jsonschema:"User name"`
    Age  int    `json:"age"  jsonschema:"User age"`
}

// Schema is automatically generated when you register the tool
handler, err := mcpio.New(
    mcpio.WithTool("my_tool", "Process user data", myToolFunc),
)
```

For dynamic schemas, use the helper functions:

```go
// Create schemas programmatically
schema := mcpio.CreateObjectSchema(
    "Dynamic input",
    map[string]string{
        "field1": "First field",
        "field2": "Second field", 
    },
    []string{"field1"}, // required fields
)

// Or use field definitions for more control
fields := []mcpio.FieldDef{
    {Name: "status", Type: "string", Required: true, Enum: []string{"active", "inactive"}},
    {Name: "count", Type: "number", Required: false},
}
dynamicSchema := mcpio.CreateDynamicSchema(fields)
```

## Comparison with Direct MCP SDK

### MCP SDK (can panic)
```go
server := mcp.NewServer(&mcp.Implementation{Name: ""}, nil) // Panics on empty name!
mcp.AddTool(server, &mcp.Tool{Name: "greet"}, handler)    // Can panic on bad types!
```

### mcp-io (error handling)
```go
handler, err := mcpio.New(
    mcpio.WithName(""), // Returns error instead of panic
    mcpio.WithTool("greet", "Say hello", greetFunc),
)
if err != nil {
    log.Printf("Configuration error: %v", err) // Graceful handling
}
```

## API Reference

### Options

- `WithName(name string)` - Set server name (required, non-empty)
- `WithVersion(version string)` - Set server version (required, non-empty)  
- `WithTool(name, desc string, fn func)` - Add typed tool
- `WithRawTool(name, desc string, schema *Schema, fn RawToolFunc)` - Add raw JSON tool
- `WithScriptTool(name, desc string, evaluator ScriptEvaluator)` - Add script tool
- `WithServer(server *mcp.Server)` - Inject custom server for testing

### Transport Methods

- `handler.ServeHTTP(w, r)` - Standard HTTP handler
- `handler.ServeSSE(w, r)` - Server-sent events transport
- `handler.ServeStdio(stdin, stdout)` - Stdio transport for CLI tools

### Error Types

- `mcpio.NewToolError(message)` - User-facing tool error
- `mcpio.ValidationError(message)` - Input validation error
- `mcpio.ProcessingError(message)` - Processing error

## Benefits

1. **Error Safety**: No panics - all configuration errors return meaningful messages
2. **Type Safety**: Compile-time validation with automatic schema generation
3. **Clean API**: Functional options pattern for readable, composable code
4. **Transport Agnostic**: Single handler works with HTTP, SSE, and stdio
5. **MCP Compatible**: Built on the official MCP SDK
6. **Testing Friendly**: Dependency injection support with `WithServer`

## Contributing

Contributions welcome! Please ensure all code examples in the README are tested in `readme_test.go`.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Related Projects

- [Official MCP SDK](https://github.com/modelcontextprotocol/go-sdk) - The underlying MCP implementation
- [MCP Specification](https://github.com/modelcontextprotocol/specification) - The Model Context Protocol specification