# mcp-io

[![Go Reference](https://pkg.go.dev/badge/github.com/robbyt/mcp-io.svg)](https://pkg.go.dev/github.com/robbyt/mcp-io)
[![Go Report Card](https://goreportcard.com/badge/github.com/robbyt/mcp-io)](https://goreportcard.com/report/github.com/robbyt/mcp-io)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A library that wraps the [Model Context Protocol (MCP) SDK](https://github.com/modelcontextprotocol/go-sdk) to provide a functional options constructor API, and better error handling and ergonomics when creating MCP servers. 

## Overview

The official MCP is simple, and very flexible. However, there are some behaviors that I didn't like: it panics when required values are missing, it uses bare structs instead of constructors, and the overall API is much more flexible than what I needed. So I wrote this package as an opinionated wrapper to add some guardrails and abstractions.

## Features

- **Graceful Error Handling**: Configuration errors return meaningful error messages instead of panicking
- **Functional Options Constructors**: Composable API using the functional options pattern
- **Type-Safe Tools**: Define MCP resources with Go generics to specify the in/out schema shapes
- **Multiple Transports**: HTTP, SSE, and stdio support through a single handler
- **Sentinel Error Types**: Errors return specific types that can be checked with `errors.Is`
- **Extensive Examples**: My main goal was to learn the MCP spec, so many examples are included

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
	handler, err := mcpio.NewHandler(
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

## Schema Flexibility (New in v0.8.0+)

The library now supports multiple ways to define tool schemas, giving you complete flexibility while optimizing for performance:

```go
// Traditional struct-based schemas (unchanged - recommended for most use cases)
mcpio.WithTool("to_upper", "Convert text to uppercase", toUpperFunc)

// Custom JSON schemas from strings (converted to optimal json.RawMessage)
mcpio.WithToolWithSchema("calculator", "Arithmetic calculator", calcFunc, &mcpio.ToolSchemas{
    InputSchema: `{
        "type": "object",
        "properties": {
            "operation": {"type": "string", "enum": ["add", "subtract", "multiply", "divide"]},
            "a": {"type": "number"},
            "b": {"type": "number"}
        },
        "required": ["operation", "a", "b"]
    }`,
    OutputSchema: `{
        "type": "object",
        "properties": {"result": {"type": "number"}},
        "required": ["result"]
    }`,
})

// Maximum performance with json.RawMessage (zero marshaling overhead)
mcpio.WithToolWithSchema("fast_processor", "High-performance processing", processorFunc, &mcpio.ToolSchemas{
    InputSchema:  json.RawMessage(`{"type":"object","additionalProperties":true}`),
    OutputSchema: json.RawMessage(`{"type":"object","properties":{"processed":{"type":"boolean"}}}`),
})

// Dynamic schemas using map[string]any
dynamicSchema := map[string]any{
    "type": "object",
    "properties": map[string]any{
        "message": map[string]any{"type": "string"},
        "repeat": map[string]any{"type": "integer", "minimum": 1, "maximum": 10},
    },
    "required": []string{"message"},
}
mcpio.WithToolWithSchema("echo", "Echo with repetition", echoFunc, &mcpio.ToolSchemas{
    InputSchema: dynamicSchema,
})
```

### Performance Hierarchy

Based on analysis of MCP SDK v0.8.0 internals, schema types are optimized in this order:

1. **`json.RawMessage`** - Zero marshaling overhead, direct wire compatibility (fastest)
2. **JSON strings** - Validated and converted to `json.RawMessage` internally
3. **`*jsonschema.Schema`** - Type-safe but requires JSON marshaling
4. **`map[string]any`** - Flexible but with map overhead (slowest)

**Recommendation**: Use `json.RawMessage` for high-performance tools, JSON strings for readability, and struct-based schemas for development convenience.

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
handler, err := mcpio.NewHandler(
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
// Create handler once
handler, err := mcpio.NewHandler(
    mcpio.WithName("my-server"),
    mcpio.WithVersion("1.0.0"),
    mcpio.WithTool("to_upper", "Convert text", toUpper),
)
if err != nil {
    log.Fatal(err)
}

// Choose your transport:

// HTTP - for standard HTTP clients
http.Handle("/mcp", handler)
log.Fatal(http.ListenAndServe(":8080", nil))

// SSE - for browser clients with server-sent events
http.Handle("/mcp-sse", http.HandlerFunc(handler.ServeSSE))
log.Fatal(http.ListenAndServe(":8080", nil))

// Stdio - for CLI tools and process communication
if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
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
handler, err := mcpio.NewHandler(
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
import "github.com/robbyt/mcp-io/schema"

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

inputSchema := schema.NewObject(
    schemaDescription, // Human-readable description of this schema
    properties,        // Map of field names to field descriptions  
    requiredFields,    // List of required field names
)

handler, err := mcpio.NewHandler(
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
handler, err := mcpio.NewHandler(
    mcpio.WithTool("my_tool", "Process user data", myToolFunc),
)
```

For schemas that can change shape, use the `schema.NewObject` helper function:

```go
import "github.com/robbyt/mcp-io/schema"

// Create schemas programmatically
s := schema.NewObject(
    "Dynamic input",
    map[string]string{
        "field1": "First field",
        "field2": "Second field", 
    },
    []string{"field1"}, // required fields
)

// Or use field definitions for more control
fields := []schema.FieldDef{
    {Name: "status", Type: "string", Required: true, Enum: []string{"active", "inactive"}},
    {Name: "count", Type: "number", Required: false},
}
dynamicSchema := schema.NewDynamic(fields)
```

## Session Capabilities

mcp-io provides access to MCP session capabilities through context-based injection. The session is automatically available in all tool, prompt, and resource handlers, allowing you to access features like elicitation, sampling, logging, and progress notifications.

### Elicitation (Interactive User Input)

Request additional information from users at runtime for configuration, preferences, or confirmation. Elicitation enables interactive workflows where tools can gather structured data dynamically.

#### Quick Example

```go
type UserConfig struct {
    Name        string `json:"name" jsonschema:"Your full name"`
    Environment string `json:"environment" jsonschema:"enum=dev,enum=prod"`
    Port        int    `json:"port" jsonschema:"minimum=1024,maximum=65535"`
}

func setupTool(ctx context.Context, _ struct{}) (map[string]any, error) {
    // Session is automatically available via context
    result, err := mcpio.ElicitTyped[UserConfig](ctx, "Enter configuration:")
    if err != nil {
        return nil, err
    }

    if result.IsAccepted() {
        var config UserConfig
        if err := result.DecodeContent(&config); err != nil {
            return nil, err
        }
        return map[string]any{"status": "configured", "config": config}, nil
    }

    return map[string]any{"status": "cancelled"}, nil
}

// Register tool - session is automatically injected
handler, err := mcpio.NewHandler(
    mcpio.WithName("interactive-server"),
    mcpio.WithTool("setup", "Interactive setup", setupTool),
)
```

#### Elicitation Methods

- **`ElicitTyped[T](ctx, message)`** - Type-safe elicitation with automatic schema generation from Go structs
- **`ElicitSimple(ctx, message, fieldName, prompt)`** - Single string field elicitation for quick inputs or confirmations

**Security Warning**: Never use elicitation for passwords, API keys, or secrets. Use only for configuration data, preferences, and non-sensitive user input.

### Sampling (LLM Integration)

[Sampling](https://modelcontextprotocol.io/specification/2025-06-18/client/sampling) allows your MCP server to delegate LLM work to the client. The server doesn't run an LLM itself—instead, it sends prompts to the client's LLM and uses the responses as helpers to build its output. This offloads expensive inference from your server to the client.

**The workflow:** Your tool receives input → builds a prompt → sends it to the client's LLM → uses the LLM's response in your final output.

#### Example: AI Dungeon Master

```go
import "github.com/robbyt/mcp-io/capabilities"

type AdventureInput struct {
    Action string `json:"action" jsonschema:"What the player does"`
}

func dungeonMaster(ctx context.Context, input AdventureInput) (map[string]any, error) {
    // Delegate storytelling to the client's LLM
    prompt := "You are a dungeon master. The player: \"" + input.Action +
              "\". Narrate what happens next in 2 sentences. Be dramatic!"

    result, err := mcpio.CreateMessage(ctx, []*capabilities.Message{{
        Role:    "user",
        Content: prompt,
    }}, 300)
    if err != nil {
        return nil, err
    }

    // Use the LLM's narrative in our response
    return map[string]any{"narrative": result.Content.Text}, nil
}
```

**Example interaction:**
```bash
# Start the server
cd examples/simple_dungeon_master
go run main.go

# In another terminal, call the tool
mcp call dungeon_master --params '{"action":"I open the mysterious door"}' http://localhost:8080/mcp
```

**Response:**
```json
{
  "narrative": "The ancient door groans open, revealing a chamber of gold coins. Suddenly, the coins coalesce into a massive golden serpent with ruby eyes!"
}
```

### Progress Notifications

Send progress updates for long-running operations to keep users informed.

```go
func processDataTool(ctx context.Context, input struct{ Files []string }) (map[string]any, error) {
    total := float64(len(input.Files))
    for i, file := range input.Files {
        mcpio.NotifyProgress(ctx, float64(i), total)
        // Process file...
    }
    mcpio.NotifyProgress(ctx, total, total) // Mark complete
    return map[string]any{"status": "done"}, nil
}
```

### Logging

Send structured log messages to the client for debugging and monitoring.

```go
func myTool(ctx context.Context, input MyInput) (MyOutput, error) {
    mcpio.LogInfo(ctx, "Processing started", map[string]any{
        "itemCount": len(input.Items),
    })

    // Do work...

    mcpio.LogDebug(ctx, "Detailed state", map[string]any{
        "processed": processed,
        "remaining": remaining,
    })

    return output, nil
}
```

Available logging functions:
- `LogDebug(ctx, message, data)` - Debug-level logs
- `LogInfo(ctx, message, data)` - Informational logs
- `LogWarn(ctx, message, data)` - Warning logs
- `LogError(ctx, message, data)` - Error logs

### Direct Session Access

For advanced use cases, access the full session capability:

```go
func advancedTool(ctx context.Context, _ struct{}) (map[string]any, error) {
    session := mcpio.GetSession(ctx)
    if session == nil {
        return nil, errors.New("no session available")
    }

    // Check capabilities
    if session.SupportsElicitation() {
        // Use elicitation...
    }
    if session.SupportsSampling() {
        // Use sampling...
    }

    // Access client capabilities for detailed information
    caps := session.ClientCapabilities()
    if caps.Roots != nil && caps.Roots.ListChanged {
        // Client supports notifications for roots list changes
        roots, _ := session.ListRoots(ctx)
        // Use roots...
    }

    // Access session info
    sessionID := session.SessionID()
    logger := session.Logger()

    return map[string]any{"sessionID": sessionID}, nil
}
```

### Learn More

- Complete elicitation example: [examples/cli_elicitation](examples/cli_elicitation/)
- Multi-step workflows: [examples/http_multistep](examples/http_multistep/)
- See [godoc](https://pkg.go.dev/github.com/robbyt/mcp-io) for full API documentation

## Migration Guide

### Upgrading to Schema Flexibility (v0.8.0+)

The new schema flexibility features are backward compatible. Existing code continues to work unchanged:

```go
// Before and after - no changes needed
mcpio.WithTool("my_tool", "Description", myToolFunc)
```

To leverage new schema options, use WithToolWithSchema:

```go
// Add custom schema options
mcpio.WithToolWithSchema("my_tool", "Description", myToolFunc, &mcpio.ToolSchemas{
    InputSchema:  `{"type":"object","properties":{"field":{"type":"string"}}}`,
    OutputSchema: outputSchema,
})
```

### Raw Tool Schema Updates

If you're using `WithRawTool`, the schema parameter now accepts `any` instead of `*jsonschema.Schema`:

```go
// Before (still works)
schema := &jsonschema.Schema{Type: "object", Properties: ...}
mcpio.WithRawTool("tool", "desc", schema, rawFunc)

// After (recommended for performance)
schemaJSON := `{"type":"object","properties":...}`
mcpio.WithRawTool("tool", "desc", schemaJSON, rawFunc)
```

### Performance Optimization

For high-performance tools, use `json.RawMessage`:

```go
mcpio.WithToolWithSchema("fast_tool", "High-performance tool", toolFunc, &mcpio.ToolSchemas{
    InputSchema:  json.RawMessage(`{"type":"object","properties":...}`),
    OutputSchema: json.RawMessage(`{"type":"object","properties":...}`),
})
```

## Comparison with Direct MCP SDK

### MCP SDK (can panic)
```go
server := mcp.NewServer(&mcp.Implementation{Name: ""}, nil) // Panics on empty name!
mcp.AddTool(server, &mcp.Tool{Name: "greet"}, handler)    // Can panic on bad types!
```

### mcp-io (error handling)
```go
handler, err := mcpio.NewHandler(
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