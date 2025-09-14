# mcp-io

[![Go Reference](https://pkg.go.dev/badge/github.com/robbyt/mcp-io.svg)](https://pkg.go.dev/github.com/robbyt/mcp-io)
[![Go Report Card](https://goreportcard.com/badge/github.com/robbyt/mcp-io)](https://goreportcard.com/report/github.com/robbyt/mcp-io)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A library that wraps the [Model Context Protocol (MCP) SDK](https://github.com/modelcontextprotocol/go-sdk) to provide a functional options constructor API, and better error handling and ergonomics when creating MCP servers. 

## Overview

The official MCP is simple, and very flexible. However, there are some behaviors that I didn't like when using it- It panics when some required values are missing, it uses bare structs instead of constructors, and the overall API is more flexible than what I needed. So I wrote those library as an opinionated wrapper to add some guardrails and abstractions.

## Features

- **Graceful Error Handling**: Configuration errors return meaningful error messages instead of panicking
- **Functional Options Constructors**: Composable API using the functional options pattern
- **Type-Safe Tools**: Define MCP resources with Go generics to specify the in/out schema shapes
- **Multiple Transports**: HTTP, SSE, and stdio support through a single handler
- **Sentinel Error Types**: Errors return specific types that can be checked with `errors.Is`

## Installation

```bash
go get github.com/robbyt/mcp-io
```

## Quick Start

Here's a simple example of creating an MCP server that exposes a tool to convert text to uppercase:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	mcpio "github.com/robbyt/mcp-io"
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
	handler, err := mcpio.NewToolHandler(
		mcpio.WithName("example-server"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
	)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	// Start HTTP server using the handler function returned by mcpio.New
	http.Handle("/mcp", handler)
	log.Printf("MCP server listening on :8080/mcp")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Testing Your Server

You can test your MCP server using the [MCP CLI tools](https://github.com/f/mcptools):

```bash
# Install the CLI tool
go install github.com/f/mcptools/cmd/mcptools@latest

# Test your HTTP server (from the example above running on :8080/mcp)
mcp tools http://localhost:8080/mcp

# Call the to_upper tool
mcp call to_upper --params '{"text":"hello world"}' http://localhost:8080/mcp

# Use different output formats
mcp tools --format json http://localhost:8080/mcp
mcp tools --format pretty http://localhost:8080/mcp
```

## Core Development Concepts

### Instantiation of the Handler

The library uses a functional options pattern for clean, composable configuration:

```go
handler, err := mcpio.NewToolHandler(
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

A single handler supports multiple transport types. Here are complete examples for each:

#### HTTP Transport

```go
// Create handler
handler, err := mcpio.NewToolHandler(
    mcpio.WithName("my-server"),
    mcpio.WithVersion("1.0.0"),
    mcpio.WithTool("to_upper", "Convert text", toUpper),
)
if err != nil {
    log.Fatal(err)
}

// Start HTTP server
http.Handle("/mcp", handler)
log.Printf("HTTP server listening on :8080/mcp")
log.Fatal(http.ListenAndServe(":8080", nil))
```

#### SSE Transport

```go
// Create handler
handler, err := mcpio.NewToolHandler(
    mcpio.WithName("my-server"),
    mcpio.WithVersion("1.0.0"),
    mcpio.WithTool("to_upper", "Convert text", toUpper),
)
if err != nil {
    log.Fatal(err)
}

// Start SSE server for browser clients
http.Handle("/mcp-sse", http.HandlerFunc(handler.ServeSSE))
log.Printf("SSE server listening on :8080/mcp-sse")
log.Fatal(http.ListenAndServe(":8080", nil))
```

#### Stdio Transport

```go
// Create handler
handler, err := mcpio.NewToolHandler(
    mcpio.WithName("my-server"),
    mcpio.WithVersion("1.0.0"),
    mcpio.WithTool("to_upper", "Convert text", toUpper),
)
if err != nil {
    log.Fatal(err)
}

// Start stdio transport for CLI tools
if err := handler.ServeStdio(os.Stdin, os.Stdout); err != nil {
    log.Fatal(err)
}
```

### Input/Output Schema Definition

Define the input/output schema required for receiving and responding to MCP tool requests, using structs. Set `jsonschema` struct tags to set additional option and guidance to the LLM for populating and working with the fields in the schema. This text will appear in the schema description, and guides the LLM to provide better input and understand the output.

```go
type MyInput struct {
    Name string `json:"name" jsonschema:"User's full name"`
    Age  int    `json:"age"  jsonschema:"User's age in years"`
}
```

```go
type CalculateInput struct {
    Operation string  `json:"operation" jsonschema:"Arithmetic operation. Specify only one: add, subtract, multiply, or divide"`
    A         float64 `json:"a" jsonschema:"First number"`
    B         float64 `json:"b" jsonschema:"Second number"`
}

type CalculateOutput struct {
    Result float64 `json:"result" jsonschema:"Calculation result"`
}

// calculate uses the CalculateInput as an input and returns CalculateOutput 
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
handler, err := mcpio.NewToolHandler(
    mcpio.WithName("calculator"),
    mcpio.WithTool("calculate", "Perform arithmetic operations", calculate),
)
if err != nil {
    log.Fatalf("Failed to register tool: %v", err)
}
```

## Advanced Features

### Raw JSON Tools

Use raw JSON tools when you need to:
- Accept arbitrary JSON structures that can't be predefined as Go structs
- Process JSON-to-JSON transformations where the structure varies
- Work with dynamic schemas determined at runtime
- Interface with external APIs that return varying JSON formats

```go
// Example: A tool that validates and reformats any JSON input
validateJSON := func(ctx context.Context, input []byte) ([]byte, error) {
    // Unmarshal to confirm it's valid JSON
    var jsonData any
    if err := json.Unmarshal(input, &jsonData); err != nil {
        return nil, mcpio.ValidationError("Invalid JSON: " + err.Error())
    }
    
    // Re-marshal back to JSON with indentation for pretty formatting
    formatted, err := json.MarshalIndent(jsonData, "", "  ")
    if err != nil {
        return nil, mcpio.ProcessingError("Failed to format JSON: " + err.Error())
    }
    
    // Return the formatted JSON wrapped in a result object
    result := map[string]any{
        "formatted_json": string(formatted),
        "valid": true,
        "size_bytes": len(input),
    }
    
    return json.Marshal(result)
}

// Define the input schema - accepts any JSON object
schemaDescription := "JSON validation input"
properties := map[string]string{
    "json_data": "Any JSON object or array to validate and format",
}
requiredFields := []string{"json_data"}

inputSchema := mcpio.CreateObjectSchema(
    schemaDescription, // Human-readable description of this schema
    properties,        // Map of field names to field descriptions  
    requiredFields,    // List of required field names
)

handler, err := mcpio.NewToolHandler(
    mcpio.WithName("json-processor"),
    mcpio.WithRawTool("validate_json", "Validate and format any JSON input", inputSchema, validateJSON),
)
if err != nil {
    log.Fatalf("Failed to create raw tool: %v", err)
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
handler, err := mcpio.NewToolHandler(
    mcpio.WithTool("my_tool", "Process user data", myToolFunc),
)
```

For schemas that can change shape, use the `CreateObjectSchema` helper function:

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
handler, err := mcpio.NewToolHandler(
    mcpio.WithName(""), // Returns error instead of panic
    mcpio.WithTool("greet", "Say hello", greetFunc),
)
if err != nil {
    log.Printf("Configuration error: %v", err) // Graceful handling
}
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Related Projects

- [Official MCP SDK](https://github.com/modelcontextprotocol/go-sdk) - The underlying MCP implementation
- [MCP Specification](https://github.com/modelcontextprotocol/specification) - The Model Context Protocol specification