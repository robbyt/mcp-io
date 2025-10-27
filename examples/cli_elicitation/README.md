# Database with Elicitation Example

An in-memory database MCP server that demonstrates elicitation - the ability for servers to pause tool execution and request additional information from users through the MCP client. This example provides a complete CRUD database with both standard operations and elicitation-enhanced operations, showing when and how to use elicitation for user confirmations and data gathering. For more complex multi-step workflows, see the http_multistep example.

## Features Demonstrated

**Complete Database**: Fully functional in-memory database with CRUD operations that works with Claude Desktop.

**Mixed Tool Types**: Demonstrates both standard tools (read_record, list_records) and elicitation-enhanced tools (create_record, update_record, delete_record).

**Typed Schemas**: Uses `ElicitTyped[T]()` for automatic JSON schema generation from Go structs for structured data input.

**Confirmation Patterns**: Shows different confirmation patterns:
- **create_record**: Elicits structured data using typed schemas
- **update_record**: Shows changes and requires "UPDATE" confirmation
- **delete_record**: Requires typing the record ID for critical operations

**Interactive Prompts**: The `database_report` prompt elicits report preferences before generating analysis prompts.

## MCP Elicitation Flow

```
1. Client calls a server tool.
2. Server realizes it needs more input -> issues an elicitation request.
3. Client prompts the user and validates their response.
4. Client returns the elicited input to the server.
5. Server resumes and completes the tool call.

┌─────────────────┐     1. tools/call("create_record", {})
│   MCP Client    │ ────────────────────────────────┐
└─────────────────┘                                 │
                                                    v
                                           ┌──────────────────┐
                                           │ MCP Server       │
                                           │ (cli_elicitation)│
                                           └──────────────────┘
                                                    │
                        2. elicitation/create       │
                        {                           │
                          message: "Please provide the details for the new record:",
                          requestedSchema: { Record schema }
                        }                           │
┌─────────────────┐                                 │
│   MCP Client    │ ◄───────────────────────────────┘
└─────────────────┘
        │
        │   3. Prompt user for input
        v
┌─────────────────┐
│      User       │
└─────────────────┘
        │
        │   4. Filled form / response
        v
┌─────────────────┐     5. elicitation response
│   MCP Client    │ ───────────────────────────────┐
└─────────────────┘     {                          │
                          action: "accept",        │
                          content: { record data } │
                        }                          v
                                           ┌──────────────────┐
                                           │ MCP Server       │
                                           │ (cli_elicitation)│
                                           └──────────────────┘
                                                   │
                        6. tools/call result       │
                        {                          │
                          status: "created",       │
                          record: { record data }  │
                        }                          │
┌─────────────────┐                                │
│   MCP Client    │ ◄──────────────────────────────┘
└─────────────────┘
```

**Key Point**: MCP elicitation enables servers to pause tool execution and request additional information from users through the MCP client. The server initiates the elicitation request during tool execution, the client mediates by prompting the user, and the server resumes processing once the response is received. The client maintains full control over user interaction and can accept, decline, or cancel elicitation requests.

**Responsibilities**:
- **Server**: Defines when and what additional input is needed during tool execution
- **Client**: Mediates the request, owns the UI/UX, validates against schema, lets the user accept/decline/cancel
- **User**: Provides structured input when prompted by the client

**Security Note**: This example includes warnings about sensitive data - elicitation should never be used for passwords, API keys, or other secrets in production environments. It's designed for configuration data, preferences, and other non-sensitive user input.

For more details on MCP elicitation, see the [official MCP specification](https://modelcontextprotocol.io/specification/2025-06-18/client/elicitation).

## Client Responsibilities

When implementing MCP clients that support elicitation, clients should:

- Display server identification when showing elicitation requests
- Present the elicitation message and required fields clearly
- Allow users to review and modify their input before submission
- Provide explicit options to decline or cancel the request

## When to Use Elicitation vs Regular Parameters

**Use Elicitation When**:
- Creating or modifying data that requires user confirmation
- Gathering information that wasn't available when the tool was called
- Implementing multi-step workflows where each step depends on user input
- Requesting confirmations for destructive or irreversible operations
- Collecting complex structured data with validation requirements

**Use Regular Tool Parameters When**:
- Information is already available from the context or previous interactions
- Performing read-only operations that don't require user confirmation
- The required data is simple and doesn't need validation
- Automating tasks where user interruption would be disruptive

**Schema Constraints**:
- Elicitation schemas must be flat objects (no nested structures)
- Supported types: string, number, boolean, enums
- String formats supported: email, uri, date, date-time
- JSONSchema validation supported: minLength, maxLength, minimum, maximum, etc.

**Best Practices**:
- Define reusable input types (like `CreateRecordInput`, `ReportConfig`) to avoid duplicating struct definitions
- Use helper functions for common validation patterns to reduce code duplication
- Use `ElicitTyped[T](ctx, message)` for type-safe elicitation with automatic schema generation
- Session is automatically available via context - no capability parameter needed
- **Important for stdio transport**: Never use `fmt.Printf` or `fmt.Println` in stdio servers - use `log.Printf` (stderr) instead to avoid corrupting the JSON-RPC protocol on stdout


## Running

```bash
make build-cli-elicitation
./bin/cli-elicitation
```

## Testing the Elicitation Server

**Important**: The `mcp` CLI tool does not support elicitation features. Elicitation requires a session-aware MCP client that can handle interactive back-and-forth communication. The CLI tool only supports stateless, one-shot operations.

**Timeout Note**: If you experience timeout errors (-32001) with the MCP Inspector, the default timeout is 10 seconds. For elicitation testing, use a longer timeout by adding `?timeout=30000` (30 seconds) to the Inspector URL.

### What you CAN test with mcp CLI:
```bash
# List available tools (works)
mcp tools ./bin/cli-elicitation

# List available prompts (works)
mcp prompts ./bin/cli-elicitation

# Test standard tools (works)
mcp call read_record --params '{"id":"test1"}' ./bin/cli-elicitation
mcp call list_records --params '{}' ./bin/cli-elicitation

# Get a prompt template (works)
mcp get-prompt database_report --params '{}' ./bin/cli-elicitation
```

### What you CANNOT test with mcp CLI:
```bash
# These will fail - elicitation requires session support
mcp call create_record --params '{}' ./bin/cli-elicitation
mcp call update_record --params '{"id":"test1","name":"New Name"}' ./bin/cli-elicitation
mcp call delete_record --params '{"id":"test1"}' ./bin/cli-elicitation
```

### To test elicitation features:
- Use an MCP client that supports elicitation (like Claude Desktop)
- Use the MCP Inspector with extended timeout: `npx @modelcontextprotocol/inspector go run -C examples/cli_elicitation .` then add `?timeout=30000` to the URL for 30 seconds
- Write integration tests with mock elicitation handlers
- Use the HTTP version with a custom client that handles elicitation requests

## Confirmation Examples

These patterns ensure safe database operations while demonstrating various elicitation techniques.

### Structured Data Elicitation
```go
// Define a reusable input type with validation constraints
type CreateRecordInput struct {
    ID     string `json:"id"     jsonschema:"description:Unique identifier (no spaces),minLength:1,maxLength:50"`
    Name   string `json:"name"   jsonschema:"description:Display name,minLength:1,maxLength:100"`
    Email  string `json:"email"  jsonschema:"format:email,description:Email address,maxLength:255"`
    Status string `json:"status" jsonschema:"description:Record status,enum:active,enum:inactive,enum:pending,enum:archived"`
    Age    int    `json:"age"    jsonschema:"description:Age in years,minimum:18,maximum:120"`
}

// Elicit structured record data using the defined type
result, err := mcpio.ElicitTyped[CreateRecordInput](ctx,
    "To create a new database record, please provide the following information:")
if err != nil {
    return nil, err
}

if result.IsAccepted() {
    var recordData CreateRecordInput
    if err := result.DecodeContent(&recordData); err != nil {
        return nil, err
    }
    // Use the structured data to create the record...
}
```

### Critical Operation Confirmation
```go
// Helper function to reduce confirmation validation duplication
func validateConfirmation(result *mcpio.ElicitationResult, expectedValue string) (bool, string) {
    if !result.IsAccepted() {
        return false, fmt.Sprintf("User %s the operation", result.Action)
    }

    content := result.GetContent()
    if content == nil {
        return false, "No confirmation provided"
    }

    confirmation, ok := content["confirm"].(string)
    if !ok || confirmation != expectedValue {
        return false, fmt.Sprintf("Invalid confirmation. Expected '%s', got '%s'", expectedValue, confirmation)
    }

    return true, ""
}

// Require typing record ID to confirm deletion
result, err := mcpio.ElicitSimple(ctx,
    fmt.Sprintf("Delete record '%s'? This cannot be undone.", recordID),
    "confirm", fmt.Sprintf("Type '%s' to confirm deletion", recordID))
if err != nil {
    return nil, err
}

// Use the helper to validate confirmation
if valid, reason := validateConfirmation(result, recordID); !valid {
    return map[string]any{"status": "cancelled", "reason": reason}, nil
}

// Proceed with deletion since confirmation is valid
```

