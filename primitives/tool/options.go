package tool

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/schema"
)

// Option is a functional option for configuring an mcp.Tool.
type Option func(*mcp.Tool) error

// WithTitle sets the human-readable display title for the tool.
//
// Per MCP spec, display precedence is: Title > Annotations.Title > Name
//
// Example:
//
//	tool.WithTitle("Weather Information Provider")
func WithTitle(title string) Option {
	return func(t *mcp.Tool) error {
		t.Title = title
		return nil
	}
}

// WithOutputSchema sets the JSON Schema defining the expected output structure.
//
// The schema can be json.RawMessage, string, *jsonschema.Schema, or map[string]any.
//
// Example:
//
//	tool.WithOutputSchema(`{"type":"object","properties":{"temp":{"type":"number"}}}`)
func WithOutputSchema(outputSchema any) Option {
	return func(t *mcp.Tool) error {
		converted, err := schema.ConvertToRawMessage(outputSchema)
		if err != nil {
			return fmt.Errorf("invalid output schema: %w", err)
		}
		t.OutputSchema = converted
		return nil
	}
}

// WithAnnotations sets behavioral hints for the tool.
//
// WARNING: Per MCP specification/2025-06-18/server/tools, clients MUST consider
// tool annotations to be untrusted unless they come from trusted servers.
//
// Example:
//
//	tool.WithAnnotations(&tool.ToolAnnotations{
//	    ReadOnlyHint:   true,
//	    IdempotentHint: true,
//	})
func WithAnnotations(annotations *ToolAnnotations) Option {
	return func(t *mcp.Tool) error {
		t.Annotations = annotations.toMCP()
		return nil
	}
}

// WithMeta sets protocol-reserved metadata.
//
// See specification/2025-06-18/basic/index#general-fields for _meta usage.
//
// Example:
//
//	tool.WithMeta(map[string]any{"version": "1.0", "author": "team"})
func WithMeta(meta map[string]any) Option {
	return func(t *mcp.Tool) error {
		t.Meta = meta
		return nil
	}
}

// WithReadOnly marks the tool as read-only (does not modify its environment).
//
// This is a convenience function that sets Annotations.ReadOnlyHint = true.
//
// Example:
//
//	tool.WithReadOnly()
func WithReadOnly() Option {
	return func(t *mcp.Tool) error {
		if t.Annotations == nil {
			t.Annotations = &mcp.ToolAnnotations{}
		}
		t.Annotations.ReadOnlyHint = true
		return nil
	}
}

// WithIdempotent marks the tool as idempotent (repeated calls have same effect).
//
// This is a convenience function that sets Annotations.IdempotentHint = true.
//
// Example:
//
//	tool.WithIdempotent()
func WithIdempotent() Option {
	return func(t *mcp.Tool) error {
		if t.Annotations == nil {
			t.Annotations = &mcp.ToolAnnotations{}
		}
		t.Annotations.IdempotentHint = true
		return nil
	}
}

// WithDestructive marks the tool as potentially destructive.
//
// This is a convenience function that sets Annotations.DestructiveHint = &true.
//
// Example:
//
//	tool.WithDestructive()
func WithDestructive() Option {
	return func(t *mcp.Tool) error {
		if t.Annotations == nil {
			t.Annotations = &mcp.ToolAnnotations{}
		}
		destructive := true
		t.Annotations.DestructiveHint = &destructive
		return nil
	}
}

// WithOpenWorld marks the tool as interacting with external entities.
//
// This is a convenience function that sets Annotations.OpenWorldHint = &true.
// Use this for tools that interact with web APIs, external databases, etc.
//
// Example:
//
//	tool.WithOpenWorld()
func WithOpenWorld() Option {
	return func(t *mcp.Tool) error {
		if t.Annotations == nil {
			t.Annotations = &mcp.ToolAnnotations{}
		}
		openWorld := true
		t.Annotations.OpenWorldHint = &openWorld
		return nil
	}
}

// WithClosedWorld marks the tool as having a closed domain of interaction.
//
// This is a convenience function that sets Annotations.OpenWorldHint = &false.
// Use this for tools that only interact with internal/local systems.
//
// Example:
//
//	tool.WithClosedWorld()
func WithClosedWorld() Option {
	return func(t *mcp.Tool) error {
		if t.Annotations == nil {
			t.Annotations = &mcp.ToolAnnotations{}
		}
		closedWorld := false
		t.Annotations.OpenWorldHint = &closedWorld
		return nil
	}
}
