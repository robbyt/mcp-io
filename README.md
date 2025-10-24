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

### Running and Testing This Example

Compile and run the example server:

```bash
go run main.go
```

The server will listen on http://localhost:8080/mcp and can be tested with the MCP Inspector CLI:

```bash
# List available tools
npx @modelcontextprotocol/inspector --cli http://localhost:8080/mcp --method tools/list

# Call the to_upper tool
npx @modelcontextprotocol/inspector --cli http://localhost:8080/mcp --method tools/call --tool-name to_upper --tool-arg text="hello world"
```

**Example MCP protocol output from tools/list:**

The `inputSchema` and `outputSchema` fields contain JSON Schemas definitions that are populated by the tool's parameters, and are typically handled by a MCP client library.

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

When a MCP client calls the `to_upper` tool with the argument `text="hello world"`, the server responds with:

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

## Development Concepts In Depth

### Handler Instantiation

The library uses a functional options pattern allowing for discoverable configuration, and future extensibility. Create a new MCP handler using `mcpio.NewHandler`, passing in options like server name, version, and tools:

```go
handler, err := mcpio.NewHandler(
    mcpio.WithName("my-server"),
    mcpio.WithVersion("1.0.0"),
    mcpio.WithTool("tool1", "Description", toolFunc1),
    mcpio.WithTool("tool2", "Description", toolFunc2),
)
```

### Input/Output Schema Definition

The easiest way to define JSON Schemas for your tools is to use annotated Go structs. The library automatically generates schemas from struct types using [`github.com/google/jsonschema-go/jsonschema`](https://pkg.go.dev/github.com/google/jsonschema-go/jsonschema). The `jsonschema` struct tag provides descriptions and constraints that guide the LLM in understanding and populating fields.

For simple descriptions, use the tag value directly. For advanced constraints like enums, ranges, or patterns, use comma-separated key=value pairs. See the [jsonschema package documentation](https://pkg.go.dev/github.com/google/jsonschema-go/jsonschema) for all supported annotations.

**Example: Division Calculator with Schema Constraints**

```go
package main

import (
    "context"
    "log"
    "math"

    mcpio "github.com/robbyt/mcp-io"

    // More guidance on using this in the next section:
    toolOption "github.com/robbyt/mcp-io/primitives/tool"
)

type DivideInput struct {
    Numerator   float64 `json:"numerator" jsonschema:"Number to be divided"`
    Denominator float64 `json:"denominator" jsonschema:"Number to divide by (cannot be zero)"`
    Precision   int     `json:"precision" jsonschema:"Decimal places for rounding,minimum=0,maximum=10,default=2"`
}

type DivideOutput struct {
    Result float64 `json:"result" jsonschema:"Division result rounded to specified precision"`
}

func divide(ctx context.Context, input DivideInput) (DivideOutput, error) {
    if input.Denominator == 0 {
        return DivideOutput{}, mcpio.NewToolError("division by zero")
    }

    result := input.Numerator / input.Denominator

    // Round to specified precision
    multiplier := math.Pow(10, float64(input.Precision))
    rounded := math.Round(result*multiplier) / multiplier

    return DivideOutput{Result: rounded}, nil
}

func main() {
    handler, err := mcpio.NewHandler(
        mcpio.WithName("calculator"),
        mcpio.WithTool("divide", "Divide two numbers with configurable precision", divide,
            toolOption.WithReadOnly(),
            toolOption.WithIdempotent(),
        ),
    )
    if err != nil {
        log.Fatalf("Failed to create handler: %v", err)
    }

    // The previous example used a HTTP transport, but you can also use stdio pipes for MCP communication:
    if err := handler.ServeStdio(context.Background(), nil, nil); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
```

### Tool Metadata and Multiple Tools Per Server

MCP servers typically expose multiple related tools via a single handler. Tool metadata provides behavioral hints per the [MCP specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools) that help LLMs and MCP clients make safer decisions. These annotations allow clients to understand tool behavior WITHOUT executing them. For example, an LLM can see that a tool is marked `WithDestructive()` and should (hopefully) act carefully when calling it. MCP Elicitation is another feature that adds a user confirmation step, and will be reviewed in a later section.

**Important**: Per the MCP spec, tool annotations are hints only and NOT guaranteed to provide faithful description of tool behavior. Clients MUST consider annotations untrusted unless from trusted servers, and should never make security decisions based solely on these hints.

The following example demonstrates a database management server with four related tools, each using appropriate metadata annotations:

```go
import (
	"github.com/robbyt/mcp-io"
	toolOption "github.com/robbyt/mcp-io/primitives/tool"
)

// Database management server exposing multiple related tools
handler, err := mcpio.NewHandler(
	mcpio.WithName("database-manager"),
	mcpio.WithVersion("1.0.0"),

	// Safe read operation - can be called repeatedly without side effects
	mcpio.WithTool("get_records", "Retrieve records from database", getRecords,
		toolOption.WithReadOnly(),
		toolOption.WithIdempotent(),
	),

	// Update operation - modifies state but safe to retry
	mcpio.WithTool("update_record", "Update existing database record", updateRecord,
		toolOption.WithIdempotent(),  // Same inputs produce same result
	),

	// Destructive operation - irreversible deletion
	mcpio.WithTool("delete_records", "Permanently delete records", deleteRecords,
		toolOption.WithDestructive(),  // Signals LLM to use caution
	),

	// Backup operation - interacts with external storage
	mcpio.WithTool("backup_database", "Backup to cloud storage", backupDatabase,
		toolOption.WithReadOnly(),     // Doesn't modify local database
		toolOption.WithOpenWorld(),    // Communicates with external services
	),
)
```

Tool annotations guide LLM decision-making per the MCP specification, helping clients understand tool behavior and make safer choices. For details on each annotation's meaning and usage, see the documented functions in [`primitives/tool/options.go`](primitives/tool/options.go).

### Transport Options

A single handler instance supports multiple transport types. Choose one transport at runtime based on your deployment needs:

- **stdio** - Process communication via standard input/output (recommended default)
- **HTTP** - Standard HTTP POST/GET requests (Streamable HTTP transport)
- **SSE** - Server-Sent Events for streaming (part of Streamable HTTP)

For details on MCP transport protocols, see the [official specification](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports).

```go
package main

import (
    "context"
    "flag"
    "log"
    "net/http"

    mcpio "github.com/robbyt/mcp-io"
)

func main() {
    transport := flag.String("transport", "stdio", "Transport type: http, sse, or stdio")
    flag.Parse()

    handler, err := mcpio.NewHandler(
        mcpio.WithName("my-server"),
        mcpio.WithVersion("1.0.0"),
        mcpio.WithTool("to_upper", "Convert text", toUpper),
    )
    if err != nil {
        log.Fatal(err)
    }

    switch *transport {
    case "http":
        // HTTP - for standard HTTP clients
        http.Handle("/mcp", handler)
        log.Printf("Starting HTTP server on :8080/mcp")
        log.Fatal(http.ListenAndServe(":8080", nil))

    case "sse":
        // SSE - for browser clients with server-sent events
        http.Handle("/mcp-sse", http.HandlerFunc(handler.ServeSSE))
        log.Printf("Starting SSE server on :8080/mcp-sse")
        log.Fatal(http.ListenAndServe(":8080", nil))

    case "stdio":
        // Stdio - for CLI tools and process communication
        log.Printf("Starting stdio transport")
        if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
            log.Fatal(err)
        }

    default:
        log.Fatalf("Unknown transport: %s", *transport)
    }
}
```

## Session Capabilities

Session capabilities enable interactive MCP tools with user confirmation, AI delegation, progress reporting, and detailed logging. These capabilities are automatically available in your tool handlers via context without additional setup. Not all MCP clients support all session capabilities, and your tools should gracefully handle unsupported features. The capabilities helpers return sentinel errors (like `mcpio.ErrElicitationNotSupported`) when the feature is available on the client, which can be checked and handled with `errors.Is`.

### Elicitation (Interactive User Input)

Request additional information from users at runtime for configuration, preferences, or confirmation. Elicitation enables interactive workflows where tools can gather structured data dynamically. Never use elicitation for passwords, API keys, or secrets. Use only for configuration data, preferences, and non-sensitive user input. 

This example shows using elicitation as a confirmation when performing destructive operations:

```go
type DeleteRecordsInput struct {
    UserID int `json:"user_id" jsonschema:"User ID whose records will be deleted"`
}

type ConfirmDeletion struct {
    Confirm string `json:"confirm" jsonschema:"Type DELETE to confirm deletion"`
}

func deleteRecords(ctx context.Context, input DeleteRecordsInput) (map[string]any, error) {
    // Preview what will be deleted
    records := getRecords(input.UserID) // Returns []Record from database

    if len(records) == 0 {
        return map[string]any{"deleted": 0, "message": "No records found for user"}, nil
    }

    // Build confirmation message showing what will be deleted
    message := fmt.Sprintf(
        "You are about to delete %d records for user ID %d:\n\n%s\n\nType DELETE to confirm.",
        len(records), input.UserID, formatRecordList(records),
    )

    // Ask user to confirm with elicitation
    result, err := mcpio.ElicitTyped[ConfirmDeletion](ctx, message)
    if err != nil {
        // Handle clients that don't support elicitation
        if errors.Is(err, mcpio.ErrElicitationNotSupported) {
            return nil, fmt.Errorf("cannot delete records: client does not support confirmation")
        }
        return nil, err
    }

    if result.IsAccepted() {
        var conf ConfirmDeletion
        if err := result.DecodeContent(&conf); err != nil {
            return nil, err
        }

        if conf.Confirm == "DELETE" {
            // Perform the actual deletion
            deleted := performDeletion(records)
            return map[string]any{
                "deleted": deleted,
                "message": fmt.Sprintf("Deleted %d records", deleted),
            }, nil
        }
    }

    return map[string]any{"status": "cancelled", "deleted": 0}, nil
}

// Register tool - session is automatically injected
handler, err := mcpio.NewHandler(
    mcpio.WithName("database-manager"),
    mcpio.WithTool("delete_records", "Delete database records with confirmation", deleteRecords,
        toolOption.WithDestructive(),
    ),
)
```

### Sampling (LLM Integration)

[Sampling](https://modelcontextprotocol.io/specification/2025-06-18/client/sampling) allows your MCP server to delegate LLM work to the client. The server doesn't run the LLM inference, it sends prompts to the client's LLM and uses the responses as helpers to build its output. This offloads expensive inference from your server to the client.

Your MCP tool receives input -> builds a prompt -> sends it to the client's LLM -> receives a response from the prompt and uses the LLM's response in your final output.

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
