# Database with Elicitation Example

A practical in-memory database MCP server that demonstrates elicitation - the ability for servers to pause tool execution and request additional information from users through the MCP client. This example provides a complete CRUD database with both standard operations and elicitation-enhanced operations, showing when and how to use elicitation for user confirmations and data gathering. For more complex multi-step workflows, see the http_multistep example.

## MCP Elicitation Flow

```
1. Client calls a server tool.
2. Server realizes it needs more input → issues an elicitation request.
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

## Features Demonstrated

**Complete Database**: Fully functional in-memory database with CRUD operations that works with Claude Desktop.

**Mixed Tool Types**: Demonstrates both standard tools (read_record, list_records) and elicitation-enhanced tools (create_record, update_record, delete_record).

**Typed Schemas**: Uses `ElicitTyped[T]()` for automatic JSON schema generation from Go structs for structured data input.

**Confirmation Patterns**: Shows different confirmation patterns:
- **create_record**: Elicits structured data using typed schemas
- **update_record**: Shows changes and requires "UPDATE" confirmation
- **delete_record**: Requires typing the record ID for critical operations

**Interactive Prompts**: The `database_report` prompt elicits report preferences before generating analysis prompts.

## Running

```bash
make build-cli-elicitation
./bin/cli-elicitation
```

## Testing the Elicitation Server

**Important**: The `mcp` CLI tool does not support elicitation features. Elicitation requires a session-aware MCP client that can handle interactive back-and-forth communication. The CLI tool only supports stateless, one-shot operations.

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
- Write integration tests with mock elicitation handlers
- Use the HTTP version with a custom client that handles elicitation requests

## Database Schema

The example uses a structured record schema for the in-memory database:

```go
type Record struct {
    ID       string    `json:"id" jsonschema:"description:Unique identifier"`
    Name     string    `json:"name" jsonschema:"description:Display name"`
    Email    string    `json:"email" jsonschema:"format:email,description:Email address"`
    Category string    `json:"category" jsonschema:"description:Record category,enum:personal,enum:business,enum:academic"`
    Created  time.Time `json:"created" jsonschema:"description:Creation timestamp"`
    Updated  time.Time `json:"updated" jsonschema:"description:Last update timestamp"`
}
```

## Operations

### Standard Tools (No Elicitation)
- **read_record**: Get a record by ID
- **list_records**: List all records with optional category filter

### Elicitation-Enhanced Tools
- **create_record**: Gathers structured record data using typed elicitation
- **update_record**: Shows changes and requires "UPDATE" confirmation
- **delete_record**: Requires typing the record ID to confirm deletion

### Interactive Prompts
- **database_report**: Elicits report preferences (format, category, sorting) before generating analysis prompts

## Confirmation Examples

### Structured Data Elicitation
```go
// Elicit structured record data
result, err := mcpio.ElicitTypedResult[RecordData](ctx, capability,
    "Please provide the details for the new record:")
if err != nil {
    return nil, err
}

if result.IsAccepted() {
    var recordData RecordData
    if err := result.DecodeContent(&recordData); err != nil {
        return nil, err
    }
    // Use the structured data...
}
```

### Critical Operation Confirmation
```go
// Require typing record ID to confirm deletion
result, err := mcpio.ElicitSimple(ctx, capability,
    fmt.Sprintf("Delete record '%s'? This cannot be undone.", recordID),
    "confirm", fmt.Sprintf("Type '%s' to confirm deletion", recordID))

if result.IsAccepted() {
    if confirmation := result.GetContent()["confirm"].(string); confirmation == recordID {
        // Proceed with deletion
    } else {
        return map[string]any{"status": "cancelled", "reason": "confirmation mismatch"}, nil
    }
}
```

These patterns ensure safe database operations while demonstrating various elicitation techniques.