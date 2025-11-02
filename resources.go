package mcpio

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResourceFunc is the function signature for user-defined resource handlers.
// The function receives a context, request context with metadata, and returns ResourceContent with an optional error.
// The URI can be accessed via reqCtx.GetIdentifier().
type ResourceFunc func(context.Context, RequestContext) (*ResourceContent, error)

// ResourceContent represents the content of a resource
type ResourceContent struct {
	Content  []byte `json:"content"`
	MIMEType string `json:"mimeType,omitempty"`
}

// createResourceHandler wraps a user function to match MCP ResourceHandler signature
func createResourceHandler(fn ResourceFunc) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		// Create request context with all MCP metadata
		reqCtx := newRequestContext(req.Params.URI, req.Params, req.Session, req.Extra)

		// Execute user function with request context
		content, err := fn(ctx, reqCtx)
		if err != nil {
			// Return all errors as protocol-level errors
			// Prompts and resources follow the same pattern: errors are protocol-level,
			// unlike tools where the SDK wraps errors for LLM visibility
			return nil, err
		}

		// Create ResourceContents based on content type
		resourceContent := &mcp.ResourceContents{
			URI:      req.Params.URI,
			MIMEType: content.MIMEType,
		}

		// Determine content type based on MIME type
		if content.MIMEType != "" && strings.HasPrefix(content.MIMEType, "text/") {
			// Text content
			resourceContent.Text = string(content.Content)
		} else {
			// Binary content
			resourceContent.Blob = content.Content
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{resourceContent},
		}, nil
	}
}
