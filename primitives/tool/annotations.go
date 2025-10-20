package tool

import "github.com/modelcontextprotocol/go-sdk/mcp"

// ToolAnnotations provides behavioral hints about a tool to help clients
// understand tool capabilities without executing them.
//
// Per MCP specification/2025-06-18/server/tools, clients MUST consider
// tool annotations to be untrusted unless they come from trusted servers.
// Annotations are hints only and not guaranteed to provide faithful
// description of tool behavior.
//
// Example:
//
//	annotations := &tool.ToolAnnotations{
//	    ReadOnlyHint:   true,
//	    IdempotentHint: true,
//	}
type ToolAnnotations struct {
	// ReadOnlyHint indicates the tool does not modify its environment.
	// Read-only tools are safe to call without side effects.
	ReadOnlyHint bool

	// IdempotentHint indicates repeated calls with the same arguments
	// have no additional effect beyond the first call.
	IdempotentHint bool

	// DestructiveHint indicates the tool may perform destructive updates
	// (delete, overwrite, etc.). Pointer allows distinguishing between
	// unset and explicitly set to false.
	DestructiveHint *bool

	// OpenWorldHint indicates the tool may interact with external entities
	// outside the client's control (web APIs, databases, etc.).
	// Pointer allows distinguishing between unset and explicitly set to false.
	//
	// Use WithOpenWorld() for true, WithClosedWorld() for false.
	OpenWorldHint *bool
}

// toMCP converts our ToolAnnotations to mcp.ToolAnnotations for internal use.
func (a *ToolAnnotations) toMCP() *mcp.ToolAnnotations {
	if a == nil {
		return nil
	}
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    a.ReadOnlyHint,
		IdempotentHint:  a.IdempotentHint,
		DestructiveHint: a.DestructiveHint,
		OpenWorldHint:   a.OpenWorldHint,
	}
}
