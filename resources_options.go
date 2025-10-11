package mcpio

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WithResource adds a resource to the handler
func WithResource(uri, description string, fn ResourceFunc) Option {
	return func(cfg *handlerConfig) error {
		if uri == "" {
			return fmt.Errorf("resource URI cannot be empty: %w", ErrEmptyValue)
		}

		registerFunc := func(server *mcp.Server) error {
			resource := &mcp.Resource{
				URI:         uri,
				Name:        uri, // Use URI as name by default
				Description: description,
			}
			handler := createResourceHandler(fn)
			server.AddResource(resource, handler)
			return nil
		}

		cfg.resources = append(cfg.resources, registerFunc)
		return nil
	}
}

// WithResourceTemplate adds a resource template to the handler
func WithResourceTemplate(uriTemplate, description string, fn ResourceFunc) Option {
	return func(cfg *handlerConfig) error {
		if uriTemplate == "" {
			return fmt.Errorf("resource template cannot be empty: %w", ErrEmptyValue)
		}

		registerFunc := func(server *mcp.Server) error {
			template := &mcp.ResourceTemplate{
				URITemplate: uriTemplate,
				Name:        uriTemplate, // Use template as name by default
				Description: description,
			}
			handler := createResourceHandler(fn) // Same handler type
			server.AddResourceTemplate(template, handler)
			return nil
		}

		cfg.resourceTemplates = append(cfg.resourceTemplates, registerFunc)
		return nil
	}
}
