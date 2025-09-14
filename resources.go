package mcpio

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResourceFunc is the function signature for user-defined resource handlers.
// The function receives a context and a URI string, and returns ResourceContent with an optional error.
type ResourceFunc func(context.Context, string) (*ResourceContent, error)

// ResourceContent represents the content of a resource
type ResourceContent struct {
	Content  []byte `json:"content"`
	MIMEType string `json:"mimeType,omitempty"`
}

// NewResourceHandler creates a new MCP handler that only supports resources and resource templates.
// For mixed resource types, use NewHandler instead.
func NewResourceHandler(opts ...Option) (*Handler, error) {
	// Pre-validate that only resources are configured
	cfg := &handlerConfig{
		name:              "mcp-server",
		version:           "1.0.0",
		tools:             make([]toolRegisterFunc, 0),
		prompts:           make([]promptRegisterFunc, 0),
		resources:         make([]resourceRegisterFunc, 0),
		resourceTemplates: make([]resourceTemplateRegisterFunc, 0),
	}

	// Apply options to check configuration
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Validate that only resources/templates are configured
	if len(cfg.tools) > 0 || len(cfg.prompts) > 0 {
		return nil, fmt.Errorf("NewResourceHandler only supports resources and resource templates; use NewHandler for mixed resource types")
	}

	// Delegate to the unified constructor
	return NewHandler(opts...)
}

// createResourceHandler wraps a user function to match MCP ResourceHandler signature
func createResourceHandler(fn ResourceFunc) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		// Execute user function
		content, err := fn(ctx, req.Params.URI)
		if err != nil {
			// Handle ToolError types for consistent error handling
			var toolErr *ToolError
			if errors.As(err, &toolErr) {
				// For resource not found errors, use the MCP protocol error
				return nil, mcp.ResourceNotFoundError(req.Params.URI)
			}
			// Protocol error
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
