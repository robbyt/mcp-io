# mcp-io

[![Go Reference](https://pkg.go.dev/badge/github.com/robbyt/mcp-io.svg)](https://pkg.go.dev/github.com/robbyt/mcp-io)
[![Go Report Card](https://goreportcard.com/badge/github.com/robbyt/mcp-io)](https://goreportcard.com/report/github.com/robbyt/mcp-io)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A library that wraps the [Model Context Protocol (MCP) SDK](https://github.com/modelcontextprotocol/go-sdk) to provide a functional options constructor API with error handling instead of panics.

## Overview

This library wraps the official [MCP SDK](https://github.com/modelcontextprotocol/go-sdk) to provide a functional options API with improved error handling. The official SDK provides flexibility through bare structs and direct initialization, which can result in runtime panics when required fields are missing. This wrapper adds validation at configuration time, returning errors instead of panicking, and uses functional options for composable configuration.

## Features

- **Graceful Error Handling**: Configuration errors return meaningful error messages instead of panicking
- **Functional Options Constructors**: Composable API using the functional options pattern
- **Type-Safe Tools**: Define MCP resources with Go generics to specify the in/out schema shapes
- **Multiple Transports**: HTTP, SSE, and stdio support through a single handler
- **Sentinel Error Types**: Errors return specific types that can be checked with `errors.Is`
- **Extensive Examples**: Includes examples demonstrating MCP features and usage patterns

## Installation

```bash
go get github.com/robbyt/mcp-io
```

## Quick Start

Here's a basic example of creating an MCP server that exposes a tool that can convert text to uppercase:

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

### Running and Testing

Run the server:

```bash
go run main.go
```

The server will start and listen on http://localhost:8080/mcp.

Test it using the MCP Inspector CLI:

```bash
# List available tools
npx @modelcontextprotocol/inspector --cli http://localhost:8080/mcp --method tools/list

# Call the to_upper tool
npx @modelcontextprotocol/inspector --cli http://localhost:8080/mcp --method tools/call --tool-name to_upper --tool-arg text="hello world"
```

**Example output from tools/list:**

```json
{
  "tools": [
    {
      "name": "to_upper",
      "description": "Convert text to uppercase",
      "inputSchema": {
        "type": "object",
        "properties": {
          "text": {
            "type": "string",
            "description": "Text to transform"
          }
        },
        "required": ["text"],
        "additionalProperties": false
      },
      "outputSchema": {
        "type": "object",
        "properties": {
          "result": {
            "type": "string",
            "description": "Transformed text"
          }
        },
        "required": ["result"],
        "additionalProperties": false
      }
    }
  ]
}
```

**Example output from tools/call:**

```json
{
  "content": [
    {
      "type": "text",
      "text": "{\"result\":\"HELLO WORLD\"}"
    }
  ],
  "structuredContent": {
    "result": "HELLO WORLD"
  }
}
```

This demonstrates how an LLM would interact with your MCP server - sending input parameters and receiving structured responses.

## Core Development Concepts

### Handler Instantiation

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

### Tool Metadata (Title and Annotations)

Tools support optional metadata for display and behavioral hints per the [MCP specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools).

#### Using Functional Options

The cleanest way to add metadata is using functional options from the `primitives/tool` package:

```go
import (
	"github.com/robbyt/mcp-io"
	toolOption "github.com/robbyt/mcp-io/primitives/tool"  // Recommended import alias
)

// Simple tool (no metadata needed)
mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper)

// Tool with display title
mcpio.WithTool("get_weather", "Get weather for location", getWeather,
	toolOption.WithTitle("Weather Information Provider"),
)

// Tool with behavioral annotations (convenience functions)
mcpio.WithTool("read_file", "Read file contents", readFile,
	toolOption.WithTitle("File Reader"),
	toolOption.WithReadOnly(),    // Does not modify environment
	toolOption.WithIdempotent(),  // Repeated calls have same effect
)

// Tool with full annotations
mcpio.WithTool("delete_record", "Delete database record", deleteRecord,
	toolOption.WithTitle("Record Deleter"),
	toolOption.WithAnnotations(&toolOption.ToolAnnotations{
		DestructiveHint: &[]bool{true}[0],  // May perform destructive updates
	}),
	toolOption.WithMeta(map[string]any{"version": "1.0"}),
)

// Combine multiple annotation hints
mcpio.WithTool("api_call", "Call external API", callAPI,
	toolOption.WithOpenWorld(),   // Interacts with external entities
	toolOption.WithIdempotent(),
)
```

**Available Convenience Options**:
- `WithReadOnly()` - Tool does not modify its environment
- `WithIdempotent()` - Repeated calls with same arguments have no additional effect
- `WithDestructive()` - Tool may perform destructive updates
- `WithOpenWorld()` - Tool may interact with external entities (web APIs, databases, etc.)
- `WithClosedWorld()` - Tool has closed domain of interaction (local/internal only)

**Display Precedence**: Per MCP spec, the display name follows this precedence: `Title` > `Annotations.Title` > `Name`

**Security Note**: Per MCP spec, clients MUST consider tool annotations untrusted unless from trusted servers. Annotations are hints only and not guaranteed to provide faithful description of tool behavior.

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

**The workflow:** Your tool receives input -> builds a prompt -> sends it to the client's LLM -> uses the LLM's response in your final output.

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

**Running the example:**

The `examples/simple_dungeon_master` directory contains a full implementation using stdio transport. This example requires a client with sampling support (like Claude Desktop) to delegate narrative generation to the client's LLM.

```bash
# List available tools
npx @modelcontextprotocol/inspector --cli go run examples/simple_dungeon_master --method tools/list

# Test the debug tool (works without sampling)
npx @modelcontextprotocol/inspector --cli go run examples/simple_dungeon_master --method tools/call --tool-name debug_gameState
```

**Output from debug_gameState:**
```json
{
  "content": [
    {
      "type": "text",
      "text": "{\"currentTurnNumber\":0,\"skillCheckRequired\":0,\"summary\":\"\",\"turnHistory\":[]}"
    }
  ],
  "structuredContent": {
    "currentTurnNumber": 0,
    "skillCheckRequired": 0,
    "summary": "",
    "turnHistory": []
  }
}
```

**Note:** The `dungeon_master` tool requires a client with sampling support. The CLI inspector will return `"client does not support sampling"`. To fully test this example, configure it in Claude Desktop or another MCP client that supports sampling.

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

## Testing Your Server

You can test your MCP server using the [MCP CLI tools](https://github.com/f/mcptools):

```bash
# Install the CLI tool
go install github.com/f/mcptools/cmd/mcptools@latest

# Test your HTTP server (from the example above running on :8080/mcp)
mcptools tools http://localhost:8080/mcp

# Call the to_upper tool
mcptools call to_upper --params '{"text":"hello world"}' http://localhost:8080/mcp

# Use different output formats
mcptools tools --format json http://localhost:8080/mcp
mcptools tools --format pretty http://localhost:8080/mcp
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

// Define schema as JSON string
inputSchema := `{
    "type": "object",
    "properties": {
        "json_data": {
            "type": "object",
            "description": "Any JSON object or array to validate and format"
        }
    },
    "required": ["json_data"]
}`

handler, err := mcpio.NewHandler(
    mcpio.WithName("json-processor"),
    mcpio.WithRawTool("validate_json", "Validate and format any JSON input", inputSchema, validateJSON),
)
if err != nil {
    log.Fatalf("Failed to create raw tool: %v", err)
}
```

### Schema Flexibility

The library accepts multiple input types for defining tool schemas with `WithRawTool`, giving you flexibility in how you author schemas while optimizing for performance:

```go
// Traditional struct-based schemas (recommended for most use cases)
// WithTool auto-generates schemas from Go types
mcpio.WithTool("to_upper", "Convert text to uppercase", toUpperFunc)

// Custom JSON schemas with manual JSON handling
// WithRawTool requires explicit schema and raw JSON processing
calcRawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
    var params map[string]any
    json.Unmarshal(input, &params)
    // ... process params ...
    return json.Marshal(result)
}
mcpio.WithRawTool("calculator", "Arithmetic calculator", `{
    "type": "object",
    "properties": {
        "operation": {"type": "string", "enum": ["add", "subtract", "multiply", "divide"]},
        "a": {"type": "number"},
        "b": {"type": "number"}
    },
    "required": ["operation", "a", "b"]
}`, calcRawFunc)

// Maximum performance with json.RawMessage (zero marshaling overhead)
mcpio.WithRawTool("fast_processor", "High-performance processing",
    json.RawMessage(`{"type":"object","additionalProperties":true}`), processorRawFunc)

// Schema using map[string]any (can be programmatically constructed)
mapBasedSchema := map[string]any{
    "type": "object",
    "properties": map[string]any{
        "message": map[string]any{"type": "string"},
        "repeat": map[string]any{"type": "integer", "minimum": 1, "maximum": 10},
    },
    "required": []string{"message"},
}
mcpio.WithRawTool("echo", "Echo with repetition", mapBasedSchema, echoRawFunc)
```

#### Performance Hierarchy

`WithRawTool` accepts multiple input types for the schema parameter. Each type has different performance characteristics based on internal conversion to `json.RawMessage`:

1. **`json.RawMessage`** - Zero marshaling overhead, returned as-is
2. **JSON strings** - Validated and converted to `json.RawMessage` (no marshaling)
3. **`*jsonschema.Schema`** - Requires JSON marshaling to convert
4. **`map[string]any`** - Requires JSON marshaling to convert

Use `json.RawMessage` for performance-critical tools, JSON strings for readability, `map[string]any` for programmatic construction, and struct-based schemas (with `WithTool`) for type safety and development convenience.

### Manual Schema Construction

For schemas that require dynamic construction or complex validation rules, define schemas as JSON strings or using the `jsonschema` package directly:

```go
import "github.com/google/jsonschema-go/jsonschema"

// Option 1: JSON string schema (recommended for readability)
schema := `{
    "type": "object",
    "properties": {
        "status": {
            "type": "string",
            "enum": ["active", "inactive"]
        },
        "count": {
            "type": "number",
            "minimum": 0
        }
    },
    "required": ["status"]
}`

// Option 2: Using jsonschema.Schema directly
schema := &jsonschema.Schema{
    Type: "object",
    Properties: map[string]*jsonschema.Schema{
        "field1": {Type: "string", Description: "First field"},
        "field2": {Type: "string", Description: "Second field"},
    },
    Required: []string{"field1"},
}

// Use with WithRawTool
handler, err := mcpio.NewHandler(
    mcpio.WithName("dynamic-server"),
    mcpio.WithRawTool("dynamic_tool", "Tool with custom schema", schema, rawFunc),
)
```

## Migration Guide

### Schema Flexibility

The schema flexibility features are backward compatible. Existing code continues to work unchanged:

```go
// WithTool continues to work with auto-generated schemas
mcpio.WithTool("my_tool", "Description", myToolFunc)
```

For custom schemas, use `WithRawTool` with raw JSON handling:

```go
// WithRawTool requires manual JSON marshaling
myRawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
    var params map[string]any
    json.Unmarshal(input, &params)
    // ... process ...
    return json.Marshal(result)
}
mcpio.WithRawTool("my_tool", "Description", `{"type":"object","properties":{"field":{"type":"string"}}}`, myRawFunc)
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

For high-performance tools, use `WithRawTool` with `json.RawMessage`:

```go
fastToolFunc := func(ctx context.Context, input []byte) ([]byte, error) {
    // Direct byte processing - no marshaling overhead
    return processBytes(input)
}
mcpio.WithRawTool("fast_tool", "High-performance tool",
    json.RawMessage(`{"type":"object","properties":...}`), fastToolFunc)
```

## Comparison with Direct MCP SDK

This library wraps the official MCP SDK to provide additional safety and convenience. Here are the key differences:

### Configuration and Error Handling

**MCP SDK**: Uses bare structs with runtime validation
```go
// Panics at runtime if name is empty
server := mcp.NewServer(&mcp.Implementation{Name: ""}, nil)
```

**mcp-io**: Uses functional options with upfront validation
```go
// Returns error at configuration time
handler, err := mcpio.NewHandler(
    mcpio.WithName(""), // error: server name cannot be empty
)
if err != nil {
    log.Printf("Configuration error: %v", err)
}
```

### Type-Safe Tools with Generics

**MCP SDK**: Manual schema creation and type assertions
```go
tool := &mcp.Tool{
    Name:        "greet",
    Description: "Greet user",
    InputSchema: &jsonschema.Schema{ /* manual schema */ },
}

handler := func(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Manual type assertions and unmarshaling
    var input map[string]any
    json.Unmarshal(req.Params.Arguments, &input)
    name := input["name"].(string)

    result := map[string]any{"greeting": "Hello " + name}
    content, _ := json.Marshal(result)

    return &mcp.CallToolResult{Content: []any{content}}, nil
}
mcp.AddTool(server, tool, handler)
```

**mcp-io**: Automatic schema generation and type safety
```go
type GreetInput struct {
    Name string `json:"name" jsonschema:"User's name"`
}

type GreetOutput struct {
    Greeting string `json:"greeting" jsonschema:"The greeting message"`
}

func greet(ctx context.Context, input GreetInput) (GreetOutput, error) {
    return GreetOutput{Greeting: "Hello " + input.Name}, nil
}

// Schema automatically generated, types checked at compile time
handler, err := mcpio.NewHandler(
    mcpio.WithName("greeter"),
    mcpio.WithTool("greet", "Greet user", greet),
)
```

### Session Capabilities

**MCP SDK**: Manual session handling and capability checking
```go
handler := func(req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Must manually extract and check session
    if req.Session == nil {
        return nil, errors.New("no session")
    }
    if req.Session.Capabilities.Sampling == nil {
        return nil, errors.New("sampling not supported")
    }
    // Use session...
}
```

**mcp-io**: Automatic session injection via context
```go
func myTool(ctx context.Context, input MyInput) (MyOutput, error) {
    // Session automatically available in context
    result, err := mcpio.CreateMessage(ctx, messages, maxTokens)
    // Error handling built-in
    return output, nil
}
```

### Transport Handling

**MCP SDK**: Separate setup for each transport type
```go
server := mcp.NewServer(impl, nil)
// HTTP requires manual handler creation
httpHandler := mcp.NewStreamableHTTPHandler(server)
http.Handle("/mcp", httpHandler)

// SSE requires different setup
sseHandler := mcp.NewSSEHandler(server)
http.Handle("/sse", sseHandler)

// Stdio requires different invocation
transport := mcp.StdioTransport{...}
transport.Serve(server)
```

**mcp-io**: Single handler for all transport types
```go
handler, err := mcpio.NewHandler(
    mcpio.WithName("my-server"),
    mcpio.WithTool("tool", "Description", toolFunc),
)

// Use same handler for any transport
http.Handle("/mcp", handler)                    // HTTP
http.Handle("/sse", http.HandlerFunc(handler.ServeSSE)) // SSE
handler.ServeStdio(ctx, os.Stdin, os.Stdout)   // Stdio
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Related Projects

- [Official MCP SDK](https://github.com/modelcontextprotocol/go-sdk) - The underlying MCP implementation
- [MCP Specification](https://github.com/modelcontextprotocol/specification) - The Model Context Protocol specification
