# MCP Capabilities: Incomplete Abstractions Analysis

## The Problem

**The core mission of mcp-io is to make the MCP SDK easier to use.** However, incomplete capability implementations actually make it **harder** to work with than using the MCP SDK directly. This document analyzes what went wrong and how to prevent it in the future.

## Case Study: NotifyProgress

### What the MCP SDK Provides

From `github.com/modelcontextprotocol/go-sdk/mcp`:

```go
type ProgressNotificationParams struct {
    Meta          `json:"_meta,omitempty"`
    ProgressToken any     `json:"progressToken"`    // REQUIRED for concurrent requests
    Message       string  `json:"message,omitempty"` // User-facing progress description
    Progress      float64 `json:"progress"`
    Total         float64 `json:"total,omitempty"`
}

func (ss *ServerSession) NotifyProgress(ctx context.Context, params *ProgressNotificationParams) error
```

**All 5 fields are part of the MCP specification.** The `ProgressToken` field is critical for associating progress notifications with the originating request in concurrent scenarios.

### What mcp-io Provides (Current)

```go
// Public API
func NotifyProgress(ctx context.Context, progress, total float64) error

// Implementation
func (s *sessionCapability) NotifyProgress(ctx context.Context, progress, total float64) error {
    params := &mcp.ProgressNotificationParams{
        Progress: progress,  // ✅ Exposed
        Total:    total,     // ✅ Exposed
        // ProgressToken: nil,  ❌ MISSING - breaks concurrent requests
        // Message: "",         ❌ MISSING - no descriptive progress
        // Meta: {},            ❌ MISSING - no metadata
    }
    return s.session.NotifyProgress(ctx, params)
}
```

### The Impact

**Users lose critical MCP functionality:**

1. **Concurrent requests fail silently**: Client can't associate progress with specific tool calls
2. **No descriptive progress**: Can't tell user "Processing file 5 of 10: report.pdf"
3. **Forced to use raw SDK**: To get full functionality, users must bypass our abstraction entirely

**This violates the library's purpose** - instead of simplifying the SDK, we've created a broken subset that's harder to debug.

## Root Cause Analysis

### Why This Happened

1. **Focused on "simple" API first** - Prioritized ease of use over completeness
2. **No MCP spec audit** - Didn't systematically compare our types against protocol.go
3. **Missing integration tests** - Didn't test with concurrent requests or real MCP clients
4. **No field-by-field review** - Implemented "the basics" without verifying we covered everything

### The Lesson

**Partial abstractions are worse than no abstraction.** When we hide complexity, we must:

1. ✅ Support ALL required protocol fields
2. ✅ Provide escape hatches for advanced use (e.g., `NotifyProgressRaw()`)
3. ✅ Test against the actual MCP specification
4. ✅ Document what we support vs. what the protocol supports

## Capability Implementation Checklist

Before shipping any MCP capability wrapper, verify:

### 1. Protocol Completeness
- [ ] Read the MCP specification for this capability
- [ ] List all fields in the MCP SDK struct
- [ ] Verify our abstraction exposes ALL required fields
- [ ] Document any intentionally omitted fields with reasoning

### 2. Context Injection
- [ ] Identify fields that should auto-inject from request context (e.g., ProgressToken)
- [ ] Extract these in tool/prompt/resource handlers
- [ ] Store in context using typed keys
- [ ] Automatically populate in capability calls

### 3. API Design
- [ ] Simple API for common case (e.g., `NotifyProgress(progress, total)`)
- [ ] Extended API for less common fields (e.g., `NotifyProgressWithMessage(progress, total, msg)`)
- [ ] Raw API escape hatch (e.g., `NotifyProgressRaw(*mcp.ProgressNotificationParams)`)

### 4. Testing
- [ ] Unit tests for field population
- [ ] Integration tests with real MCP client
- [ ] Concurrent request scenarios
- [ ] Verify actual wire protocol matches MCP spec

### 5. Documentation
- [ ] List all supported fields
- [ ] List all unsupported fields (if any) with workarounds
- [ ] Provide examples for common and advanced use cases
- [ ] Link to MCP specification

## Audit: Other Capabilities

We should audit ALL capability abstractions using this checklist:

### NotifyProgress
**Status**: ❌ Incomplete (2 of 5 fields)
**Missing**: ProgressToken, Message, Meta
**Priority**: High - breaks concurrent requests
**Fix**: See [TODO.md](TODO.md)

### CreateMessage (Sampling)
**Status**: ⚠️ Needs Review
**Check**:
- [ ] Does `MessageResult` include Model, StopReason fields from MCP SDK?
- [ ] Do we expose temperature, topP, topK sampling parameters?
- [ ] Can users specify systemPrompt, stopSequences?
- [ ] Is maxTokens properly validated?

**MCP SDK provides**:
```go
type CreateMessageParams struct {
    Messages         []*SamplingMessage
    ModelPreferences *ModelPreferences
    SystemPrompt     string
    MaxTokens        int64
    Temperature      *float64
    TopP             *float64
    TopK             *int64
    StopSequences    []string
    Metadata         map[string]any
}

type CreateMessageResult struct {
    Content    Content
    Model      string    // ❓ Do we return this?
    Role       Role
    StopReason string    // ❓ Do we return this?
}
```

**Our abstraction**:
```go
// Simple API - good for common case
func CreateMessage(ctx context.Context, messages []*Message, maxTokens int) (*MessageResult, error)

// But do we expose advanced parameters anywhere?
```

**Action**: Verify we return all fields from CreateMessageResult and provide API for CreateMessageParams fields.

### Elicitation
**Status**: ⚠️ Needs Review
**Check**:
- [ ] Do we support all ElicitParams fields?
- [ ] Do we return all ElicitResult fields?
- [ ] Can users inspect rejection reasons?

### ListRoots
**Status**: ⚠️ Needs Review
**Check**:
- [ ] Do we expose all Root fields (URI, Name)?
- [ ] Do we handle RootsCapabilities.ListChanged?

### Logging
**Status**: ⚠️ Needs Review
**Check**:
- [ ] Do we support all LoggingLevel values?
- [ ] Can users attach arbitrary data fields?
- [ ] Do we respect client's minimum log level?

## The Path Forward

### Immediate Actions (Current PR)
1. ✅ Document NotifyProgress as work-in-progress
2. ✅ Update README.md with warnings and limitations
3. ✅ Add implementation plan to TODO.md
4. ✅ Create this analysis document (caps.md)

### Next PR (High Priority)
1. Fix NotifyProgress implementation
2. Add comprehensive tests with concurrent requests
3. Verify wire protocol matches MCP spec

### Future Work
1. Audit ALL capabilities using the checklist above
2. Add integration tests with real MCP clients (not just mocks)
3. Consider adding protocol compliance test suite
4. Document our "completeness guarantee" for capabilities

## Design Principles Going Forward

### 1. Complete Before Simple
Don't ship partial implementations. Better to:
- Ship nothing (users can use SDK directly)
- Ship complete with verbose API (can simplify later)
- Ship simple + raw escape hatch

**Never** ship simple-but-broken.

### 2. Protocol First, API Second
Design process should be:
1. Read MCP specification
2. List all protocol fields
3. Design API that exposes ALL fields (even if verbose)
4. Add convenience wrappers on top
5. Test against actual protocol

### 3. Test Real Scenarios
Mock tests aren't enough. We need:
- Integration tests with real MCP clients
- Concurrent request scenarios
- Protocol wire format verification
- Tests that fail when SDK adds new required fields

### 4. Document Explicitly
Every capability documentation should state:
- ✅ What we support
- ❌ What we don't support (if anything)
- 🔧 How to use raw SDK for unsupported features
- 📖 Link to MCP specification

## Conclusion

**The purpose of mcp-io is to make the MCP SDK easier, not harder.** Incomplete abstractions that hide required protocol fields violate this mission.

We need to be more thorough in our implementations and more honest in our documentation. If we can't support a feature completely, we should either:
1. Support it completely (even if verbose), or
2. Don't wrap it at all and let users use the SDK directly

Half-measures make the library harder to use, not easier.

---

**Author Notes**: This document should be updated as we discover and fix other incomplete capability implementations. The checklist should evolve based on what we learn from each fix.
