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
- **Interface-Based Parameters**: Functions accept RequestContext interface for dependency injection and testing
- **Multiple Transports**: Streamable HTTP and stdio support through a single handler
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
func toUpper(ctx context.Context, toolCtx mcpio.RequestContext, input TextInput) (TextOutput, error) {
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

A slightly more complex version of this example is available in `examples/cli_simple/main.go`. Run the example with an HTTP transport:

```bash
go run ./examples/cli_simple/ --listen :8080
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

func divide(ctx context.Context, toolCtx mcpio.RequestContext, input DivideInput) (DivideOutput, error) {
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
    if err := handler.Run(context.Background()); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
```

### Request Context Interface

Handler functions receive a `RequestContext` interface parameter for accessing request metadata and session capabilities:

```go
type RequestContext interface {
    GetSession() *capabilities.Session  // Access session capabilities
    GetIdentifier() string              // Tool name, prompt name, or resource URI
    GetTokenInfo() *auth.TokenInfo      // OAuth token information
    GetHeaders() http.Header            // HTTP headers from request
    GetMeta() map[string]any            // Request metadata
}
```

The MCP SDK passes concrete request types to handlers, requiring functions to depend on SDK struct layout. This library uses an interface instead, decoupling handler signatures from SDK implementation details. The interface approach enables dependency injection for testing, where mock implementations satisfy the interface contract without SDK types.

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
- **Streamable HTTP** - HTTP transport supporting both request/response and streaming (MCP 2025-03-26 spec)

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
    transport := flag.String("transport", "stdio", "Transport type: http or stdio")
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
        // Streamable HTTP - for HTTP clients (supports both request/response and streaming)
        http.Handle("/mcp", handler)
        log.Printf("Starting Streamable HTTP server on :8080/mcp")
        log.Fatal(http.ListenAndServe(":8080", nil))

    case "stdio":
        // Stdio - for CLI tools and process communication
        log.Printf("Starting stdio transport")
        if err := handler.Run(context.Background()); err != nil {
            log.Fatal(err)
        }

    default:
        log.Fatalf("Unknown transport: %s", *transport)
    }
}
```

## Session Capabilities

Session capabilities enable interactive MCP tools with user confirmation, AI delegation, progress reporting, and detailed logging. These capabilities are automatically available in your tool handlers via context without additional setup. Not all MCP clients support all session capabilities, and your tools should gracefully handle unsupported features. The capabilities helpers return sentinel errors (like `mcpio.ErrElicitationNotSupported`) when the feature is available on the client, which can be checked and handled with `errors.Is`.

See [examples/](examples/) directory for complete working examples of this library.

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

func deleteRecords(ctx context.Context, toolCtx mcpio.RequestContext, input DeleteRecordsInput) (map[string]any, error) {
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

    // Create elicitor and ask user to confirm
    elicitor := mcpio.NewElicitor(toolCtx)
    result, err := mcpio.ElicitTyped[ConfirmDeletion](ctx, elicitor, message)
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

[Sampling](https://modelcontextprotocol.io/specification/2025-06-18/client/sampling) allows your MCP server to request LLM generations from the client without needing API keys. The client maintains control over which models to use and whether to allow the request. This enables agentic behaviors where your server can make LLM calls nested inside tool execution to build more intelligent responses.

**Sampling flow:**

1. MCP tool receives input
2. Tool builds sampling messages
3. Server sends request to client using the context
4. Client gets user approval (optional)
5. Client forwards the completion request to the LLM
6. LLM generates response
7. Tool uses response in its output

#### Example: AI Dungeon Master

```go
import (
    "github.com/robbyt/mcp-io/capabilities"
    "github.com/robbyt/mcp-io/capabilities/sampling"
)

type AdventureInput struct {
    Action string `json:"action" jsonschema:"What the player does"`
}

func dungeonMaster(ctx context.Context, toolCtx mcpio.RequestContext, input AdventureInput) (map[string]any, error) {
    session := toolCtx.GetSession()
    if session == nil {
        return nil, fmt.Errorf("no session available")
    }

    // Delegate storytelling to the client's LLM
    prompt := "You are a dungeon master. The player: \"" + input.Action +
              "\". Narrate what happens next in 2 sentences. Be dramatic!"

    result, err := session.CreateMessage(ctx, []*sampling.Message{{
        Role:    "user",
        Content: prompt,
    }}, sampling.WithMaxTokens(300))
    if err != nil {
        return nil, err
    }

    // Use the LLM's narrative in our response
    return map[string]any{"narrative": result.Content.Text}, nil
}
```

**Running the example:**

The `examples/simple_dungeon_master` directory contains a full implementation using stdio transport. This example requires a client with sampling support to delegate narrative generation to the client's LLM.

```bash
# List available tools
npx @modelcontextprotocol/inspector --cli go run -C examples/simple_dungeon_master . --method tools/list

# Test the debug tool (works without sampling)
npx @modelcontextprotocol/inspector --cli go run -C examples/simple_dungeon_master . --method tools/call --tool-name debug_gameState
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

**Note:** The `dungeon_master` tool requires a client with sampling capability. The CLI inspector will return `"client does not support sampling"`. To fully test this example, configure it in Claude Desktop or another MCP client that supports sampling.

### Progress Notifications

Send progress updates to the MCP client during long-running operations using functional options to control notification behavior.

**Basic Progress** (no options):
```go
func backgroundTask(ctx context.Context, toolCtx mcpio.RequestContext, input TaskInput) (TaskOutput, error) {
    session := toolCtx.GetSession()
    total := len(input.Items)

    for i, item := range input.Items {
        session.NotifyProgress(ctx, float64(i+1), float64(total))
        // Process item...
    }
    return output, nil
}
```

**Request-Specific Progress** (with token and message):
```go
func processBatch(ctx context.Context, toolCtx mcpio.RequestContext, input struct{ Files []string }) (map[string]any, error) {
    session := toolCtx.GetSession()
    token := toolCtx.GetProgressToken()
    total := len(input.Files)

    for i, file := range input.Files {
        session.NotifyProgress(ctx, float64(i+1), float64(total),
            capabilities.WithProgressToken(token),
            capabilities.WithProgressMessage(fmt.Sprintf("Processing %s (%d/%d)", file, i+1, total)))
        // Process file...
    }

    session.NotifyProgress(ctx, float64(total), float64(total), capabilities.WithProgressToken(token))
    return map[string]any{"status": "done", "processed": total}, nil
}
```

**Available Options:**
- `capabilities.WithProgressToken(token)` - Associate notification with specific request for concurrent tracking (token is int or string)
- `capabilities.WithProgressMessage(message)` - Add descriptive message
- `capabilities.WithProgressMeta(meta)` - Add custom metadata

### Logging

⚠️ **Work in Progress**: The current logging API is incomplete and missing optional MCP protocol fields (`Logger`, `Meta`). Single-logger tools work fine, but multi-subsystem tools cannot tag logs by source (e.g., "database", "auth", "cache").

Send structured log messages to the client for debugging and monitoring.

```go
func myTool(ctx context.Context, toolCtx mcpio.RequestContext, input MyInput) (MyOutput, error) {
    session := toolCtx.GetSession()

    session.LogInfo(ctx, "Processing started", map[string]any{
        "itemCount": len(input.Items),
    })

    // Do work...

    session.LogDebug(ctx, "Detailed state", map[string]any{
        "processed": processed,
        "remaining": remaining,
    })

    return output, nil
}
```

**Note**: Future releases will add functional options to support logger names and metadata for complex multi-subsystem tools.

### Request Metadata Access

Access request-specific metadata including OAuth tokens, HTTP headers, and resource identifiers through the `ToolContext` parameter. This metadata is automatically injected when handlers are invoked by MCP clients.

**Security Note**: MCP logs (via `session.LogInfo()`, `session.LogDebug()`, etc.) are sent to the LLM as context. Never log sensitive data like OAuth tokens, session IDs, or internal identifiers to MCP logs. Use a separate logging backend (slog, standard logger, metrics system) for audit trails and analytics.

Available methods on `toolCtx`:
- `toolCtx.GetIdentifier()` - Returns the tool name, prompt name, or resource URI for the current request
- `toolCtx.GetTokenInfo()` - Returns OAuth token information if the client provided authentication
- `toolCtx.GetHeaders().Get(key)` - Returns a specific HTTP header value from the request
- `toolCtx.GetHeaders()` - Returns all HTTP headers from the request

**Example: Accessing request context**

```go
import "log/slog"

func myTool(ctx context.Context, toolCtx mcpio.RequestContext, input MyInput) (MyOutput, error) {
    // Get the tool/prompt/resource identifier
    identifier := toolCtx.GetIdentifier()

    // Read custom HTTP headers from the client
    clientVersion := toolCtx.GetHeaders().Get("X-Client-Version")
    deploymentEnv := toolCtx.GetHeaders().Get("X-Deployment-Env")

    // Use backend logger for analytics/audit (NOT MCP logs which go to LLM)
    slog.Info("Request received",
        "identifier", identifier,
        "clientVersion", clientVersion,
        "env", deploymentEnv,
    )

    // Process request...
    return output, nil
}
```

### Session Interface Access

Access the `Session` interface directly via `toolCtx` to check capabilities, call session methods, or access features not yet available through convenience helpers.

**Note**: This returns mcp-io's `Session` interface, NOT the underlying MCP SDK's `*mcp.ServerSession`. You're still using mcp-io's abstraction layer, which means you're subject to the same incomplete implementations (missing Logger, ProgressToken, Meta fields, etc.). For direct MCP SDK access, use `handler.GetServer()` to access the raw `*mcp.Server`.

```go
func advancedTool(ctx context.Context, toolCtx mcpio.RequestContext, _ struct{}) (map[string]any, error) {
    session := toolCtx.GetSession()
    if session == nil {
        return nil, errors.New("no session available")
    }

    // Check capabilities before using features
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
        roots, err := session.ListRoots(ctx)
        if err == nil {
            // Use roots...
            _ = roots
        }
    }

    // Access session methods directly
    sessionID := session.SessionID()

    return map[string]any{"sessionID": sessionID}, nil
}
```

### Completion (Autocomplete)

Provide autocomplete suggestions for prompt arguments and resource URIs. Completion handlers can use context from previously-resolved arguments to provide smarter, dependency-aware suggestions. See [examples/cli_completion/](examples/cli_completion/) for usage.

## Advanced Features

### Raw JSON Tools

Raw JSON tools work with `[]byte` directly instead of typed Go structs, providing flexibility for dynamic data structures without automatic validation. Specifying the input schema is required; output schema is optional.

**When to use raw tools (`WithRawTool`):**
- Accept arbitrary JSON structures that can't be predefined as Go structs
- Process JSON-to-JSON transformations where the structure varies
- Work with dynamic schemas determined at runtime
- Interface with external APIs that return varying JSON formats
- Need maximum flexibility and willing to handle validation yourself

**When NOT to use raw tools (use `WithTool[In, Out]` instead):**
- Data structures can be defined as Go structs
- Want compile-time type safety
- Want automatic input/output validation against schemas
- Prefer catching schema mismatches early rather than at runtime

```go
import (
    mcpio "github.com/robbyt/mcp-io"
    "github.com/robbyt/mcp-io/primitives/tool"
)

// Example: A tool that validates and reformats any JSON input
validateJSON := func(ctx context.Context, toolCtx mcpio.RequestContext, input []byte) ([]byte, error) {
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

// Define input schema (required) - tells LLM what to send
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

// Define output schema (optional) - tells LLM what to expect back
outputSchema := `{
    "type": "object",
    "properties": {
        "formatted_json": {"type": "string", "description": "Pretty-printed JSON"},
        "valid": {"type": "boolean", "description": "Whether input was valid JSON"},
        "size_bytes": {"type": "integer", "description": "Size of input in bytes"}
    }
}`

handler, err := mcpio.NewHandler(
    mcpio.WithName("json-processor"),
    mcpio.WithRawTool("validate_json", "Validate and format any JSON input", inputSchema, validateJSON,
        tool.WithOutputSchema(outputSchema),  // Optional but recommended for LLM understanding
    ),
)
if err != nil {
    log.Fatalf("Failed to create raw tool: %v", err)
}
```

### Schema Type Options

The `WithRawTool` option (introduced in the [Raw JSON Tools](#raw-json-tools) section above) accepts schemas in multiple formats. Choose the format that best fits your use case:

```go
// Option 1: JSON string (for readability, shown in previous section)
schemaJSON := `{
    "type": "object",
    "properties": {
        "operation": {"type": "string", "enum": ["add", "subtract", "multiply", "divide"]},
        "a": {"type": "number"},
        "b": {"type": "number"}
    },
    "required": ["operation", "a", "b"]
}`
mcpio.WithRawTool("calculator", "Arithmetic calculator", schemaJSON, calcRawFunc)

// Option 2: map[string]any (for programmatic/dynamic schema construction)
dynamicSchema := map[string]any{
    "type": "object",
    "properties": map[string]any{
        "message": map[string]any{"type": "string"},
        "repeat": map[string]any{"type": "integer", "minimum": 1, "maximum": 10},
    },
    "required": []string{"message"},
}
mcpio.WithRawTool("echo", "Echo with repetition", dynamicSchema, echoRawFunc)

// Option 3: json.RawMessage (best performance)
jRaw := json.RawMessage(`{
    "type": "object",
    "properties": {
        "data": {"type": "string"},
        "count": {"type": "integer", "minimum": 1}
    },
    "required": ["data"]
}`)
mcpio.WithRawTool("fast_processor", "High-performance processing", jRaw, processorRawFunc)
```

## Comparison with Direct MCP SDK

This library wraps the official MCP SDK to provide additional safety and convenience. Here are the key differences:

### Configuration and Error Handling

**MCP SDK**: Panics on invalid configuration
```go
// Panics at runtime with: panic("nil Implementation")
server := mcpSDK.NewServer(nil, nil)
```

**mcp-io**: Returns errors on invalid configuration
```go
// Returns error at configuration time
handler, err := mcpio.NewHandler(
    mcpio.WithServer(nil), // error: server cannot be nil
)
if err != nil {
    log.Printf("Configuration error: %v", err)
}
```

### Simplified Handler Signatures

Both libraries support type-safe tools with automatic schema generation. The key difference is handler signature complexity.

**MCP SDK**: Explicit access to request metadata
```go
type GreetInput struct {
    Name string `json:"name" jsonschema:"User's name"`
}

type GreetOutput struct {
    Greeting string `json:"greeting" jsonschema:"The greeting message"`
}

// Full access to request context, OAuth tokens, HTTP headers
func greet(ctx context.Context, req *mcpSDK.CallToolRequest, input GreetInput) (
    *mcpSDK.CallToolResult, GreetOutput, error,
) {
    // Access OAuth token for authorization
    token := req.Extra.TokenInfo
    // Access HTTP headers
    headers := req.Extra.Header
    // Access tool name
    toolName := req.Params.Name

    return nil, GreetOutput{Greeting: "Hello " + input.Name}, nil
}

mcpSDK.AddTool(server, tool, greet) // Schema auto-generated from types
```

**mcp-io**: Simplified signature with RequestContext parameter
```go
// Clean signature with toolCtx parameter for metadata access
func greet(ctx context.Context, toolCtx mcpio.RequestContext, input GreetInput) (GreetOutput, error) {
    // Session available via toolCtx, if needed
    session := toolCtx.GetSession()

    return GreetOutput{Greeting: "Hello " + input.Name}, nil
}

handler, err := mcpio.NewHandler(
    mcpio.WithName("greeter"),
    mcpio.WithTool("greet", "Greet user", greet), // Schema auto-generated
)
```

### Request Context Pattern

**MCP SDK**: Functions receive concrete request types
```go
func handler(ctx context.Context, req *mcpSDK.CallToolRequest, input T) (*mcpSDK.CallToolResult, T, error) {
    token := req.Extra.TokenInfo
    session := req.Session
    toolName := req.Params.Name
    // ...
}
```

**mcp-io**: Functions receive an interface
```go
func handler(ctx context.Context, toolCtx mcpio.RequestContext, input T) (T, error) {
    token := toolCtx.GetTokenInfo()
    session := toolCtx.GetSession()
    toolName := toolCtx.GetIdentifier()
    // ...
}
```

The interface decouples handlers from SDK struct layout, enabling dependency injection and simpler testing with mock implementations.

**Comparison**:

| Feature | MCP SDK | mcp-io (current) | mcp-io (planned) |
|---------|---------|------------------|------------------|
| Auto-generate schemas | ✅ Yes | ✅ Yes | ✅ Yes |
| Simple handler signature | ⚠️ 3 params, 3 returns | ✅ 3 params | ✅ 3 params |
| Access session capabilities | ✅ Via request param | ✅ Via toolCtx | ✅ Via toolCtx |
| Access OAuth tokens | ✅ Via `req.Extra.TokenInfo` | ✅ Via `toolCtx.GetTokenInfo()` | N/A |
| Access HTTP headers | ✅ Via `req.Extra.Header` | ✅ Via `toolCtx.GetHeaders()` | N/A |
| Access tool name | ✅ Via `req.Params.Name` | ✅ Via `toolCtx.GetIdentifier()` | N/A |
| Control output format | ✅ Via `CallToolResult` | ❌ Auto-wrapped | ⚠️ Raw handler option |
| Escape hatch to MCP SDK | N/A | ⚠️ Via `GetServer()` | ✅ `WithRawToolHandler` |


Request metadata (OAuth tokens, HTTP headers, tool/prompt/resource identifiers) is accessible via the `toolCtx` parameter: `toolCtx.GetTokenInfo()`, `toolCtx.GetHeaders().Get(key)`, `toolCtx.GetHeaders()`, and `toolCtx.GetIdentifier()`. See [Request Metadata Access](#request-metadata-access) for details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Related Projects

- [Official MCP SDK](https://github.com/modelcontextprotocol/go-sdk) - The underlying MCP implementation
- [MCP Specification](https://github.com/modelcontextprotocol/specification) - The Model Context Protocol specification
