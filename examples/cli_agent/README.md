# AI Code Agent Example

An MCP server that demonstrates **sampling** - the ability for servers to request the client's LLM to generate responses. This example shows how to build AI-powered agents that leverage the client's language model for code analysis and improvement.

## What is Sampling?

Sampling is an MCP capability that allows servers to send messages to the client's LLM and receive AI-generated responses. This enables servers to:
- Analyze and understand user data using AI
- Generate intelligent suggestions and improvements
- Build multi-turn agent workflows
- Leverage the client's model without running their own LLM

## Features Demonstrated

**AI Code Analysis**: Uses `CreateMessage()` to send code to the client's LLM for analysis, detecting bugs, security issues, and performance problems.

**Code Improvement**: Demonstrates focused AI-powered code improvements with customizable focus areas (performance, security, readability, testing).

**Session Capability Checking**: Shows how to detect if the client supports sampling using `SupportsSampling()`.

**Progress Notifications**: Demonstrates `NotifyProgress()` for long-running AI operations.

**Structured Logging**: Uses `LogInfo()` and `LogError()` to send structured logs to the client.

**Graceful Fallbacks**: Handles cases where sampling is not available, providing informative responses.

## Sampling Flow

```
┌─────────────────┐     1. tools/call("analyze_code", {code: "..."})
│   MCP Client    │ ────────────────────────────────────────────┐
└─────────────────┘                                             │
                                                                v
                                                       ┌──────────────────┐
                                                       │   MCP Server     │
                                                       │   (cli_agent)    │
                                                       └──────────────────┘
                                                                │
                        2. sampling/createMessage              │
                        {                                      │
                          messages: [{                         │
                            role: "user",                      │
                            content: "Analyze this code..."    │
                          }],                                  │
                          maxTokens: 2000                      │
                        }                                      │
┌─────────────────┐                                            │
│   MCP Client    │ ◄──────────────────────────────────────────┘
│   + LLM         │
└─────────────────┘
        │
        │   3. Client sends to its LLM
        v
┌─────────────────┐
│   Client's LLM  │
└─────────────────┘
        │
        │   4. LLM generates analysis
        v
┌─────────────────┐     5. sampling response
│   MCP Client    │ ───────────────────────────────────────────┐
└─────────────────┘     {                                      │
                          role: "assistant",                   │
                          content: {                           │
                            text: "Analysis: This code..."     │
                          }                                    │
                        }                                      v
                                                       ┌──────────────────┐
                                                       │   MCP Server     │
                                                       │   (cli_agent)    │
                                                       └──────────────────┘
                                                                │
                        6. tools/call result                   │
                        {                                      │
                          analysis: "...",                     │
                          suggestions: [...],                  │
                          issues_found: 2                      │
                        }                                      │
┌─────────────────┐                                            │
│   MCP Client    │ ◄──────────────────────────────────────────┘
└─────────────────┘
```

**Key Points**:
- The server initiates the sampling request during tool execution
- The client sends the messages to its LLM and manages the interaction
- The server receives the LLM's response and continues processing
- The client controls which LLM is used and how tokens are managed

## Usage Example

```go
func analyzeTool(ctx context.Context, input AnalyzeInput) (AnalyzeOutput, error) {
    session := mcpio.GetSession(ctx)
    if session == nil {
        return AnalyzeOutput{}, fmt.Errorf("no session available")
    }

    // Check if client supports sampling
    if !session.SupportsSampling() {
        return AnalyzeOutput{
            Analysis:     "Sampling not supported by client",
            SamplingUsed: false,
        }, nil
    }

    // Send message to client's LLM
    result, err := mcpio.CreateMessage(ctx, []*mcpio.Message{{
        Role:    "user",
        Content: "Analyze this code for bugs: " + input.Code,
    }}, 2000)
    if err != nil {
        return AnalyzeOutput{}, fmt.Errorf("sampling failed: %w", err)
    }

    return AnalyzeOutput{
        Analysis:     result.Content.Text,
        SamplingUsed: true,
    }, nil
}
```

## Building an Agent

This example demonstrates key patterns for building AI agents with MCP:

1. **Capability Detection**: Always check `SupportsSampling()` before using sampling
2. **Graceful Fallback**: Provide useful responses even when sampling isn't available
3. **Progress Updates**: Keep users informed during long-running AI operations
4. **Structured Logging**: Help users debug and monitor agent behavior
5. **Error Handling**: Handle sampling failures gracefully

## Running

```bash
make build-cli-agent
./bin/cli-agent
```

## Testing

The example includes integration tests that use mock sessions to test:
- Successful sampling with AI responses
- Fallback behavior when sampling is not supported
- Error handling and edge cases
- Progress notifications and logging

Run tests:
```bash
go test ./examples/cli_agent/...
```

## Client Requirements

To use sampling features, your MCP client must:
- Support the `sampling` capability in the MCP specification
- Have access to an LLM for generating responses
- Implement the `sampling/createMessage` request handler

Clients that support sampling include:
- Claude Desktop (Anthropic)
- Custom clients built with the MCP SDK that include LLM integration

## When to Use Sampling

**Use Sampling When**:
- You need AI-powered analysis or generation capabilities
- You want to leverage the client's model without running your own
- You're building interactive agents that need to understand user data
- You need intelligent suggestions based on context

**Don't Use Sampling When**:
- You have deterministic logic that doesn't need AI
- You need guaranteed, consistent responses
- The client might not have LLM access
- You need real-time responses (sampling can be slower)

## Security Considerations

- Never send sensitive data (passwords, API keys, secrets) in sampling requests
- Be aware that sampling sends data to the client's LLM
- The client controls which LLM is used and how data is processed
- Implement appropriate data filtering before sending to sampling
- Consider rate limiting and token usage in production

## See Also

- Main README: [Session Capabilities](../../README.md#session-capabilities)
- Elicitation Example: [cli_elicitation](../cli_elicitation/)
- Multi-step Workflows: [http_multistep](../http_multistep/)
