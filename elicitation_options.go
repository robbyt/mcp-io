package mcpio

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SessionAwareToolFunc is a tool function that has access to elicitation capability.
// Use this type when your tool needs to request additional information from users
// during execution. The ElicitationCapability parameter provides access to ElicitTyped,
// ElicitSimple, and other elicitation methods.
//
// Example signature:
//
//	func myTool(ctx context.Context, capability ElicitationCapability, input MyInput) (MyOutput, error)
type SessionAwareToolFunc[TIn, TOut any] func(context.Context, ElicitationCapability, TIn) (TOut, error)

// SessionAwarePromptFunc is a prompt function that has access to elicitation capability.
// Use this type when your prompt needs to gather additional context or requirements
// from users to generate more targeted responses.
//
// Example signature:
//
//	func myPrompt(ctx context.Context, capability ElicitationCapability, args map[string]any) (*PromptResult, error)
type SessionAwarePromptFunc func(context.Context, ElicitationCapability, map[string]any) (*PromptResult, error)

// SessionAwareResourceFunc is a resource function that has access to elicitation capability.
// Use this type when your resource needs to request authentication credentials,
// filter preferences, or other parameters to customize the resource content.
//
// Example signature:
//
//	func myResource(ctx context.Context, capability ElicitationCapability, uri string) (*ResourceContent, error)
type SessionAwareResourceFunc func(context.Context, ElicitationCapability, string) (*ResourceContent, error)

// WithSessionTool adds a session-aware tool with elicitation capability access.
// Session-aware tools can elicit additional information from users during execution,
// enabling interactive workflows that gather configuration, preferences, or inputs dynamically.
//
// Example:
//
//	type DeployConfig struct {
//	    Environment string `json:"environment" jsonschema:"enum=dev,enum=staging,enum=prod"`
//	    Region      string `json:"region" jsonschema:"description=AWS region"`
//	    AutoScale   bool   `json:"autoScale" jsonschema:"description=Enable auto-scaling"`
//	}
//
//	func deployApp(ctx context.Context, capability ElicitationCapability, input struct{}) (map[string]any, error) {
//	    // Elicit deployment configuration from user
//	    result, err := ElicitTypedResult[DeployConfig](ctx, capability, "Enter deployment configuration:")
//	    if err != nil {
//	        return nil, err
//	    }
//
//	    if !result.IsAccepted() {
//	        return map[string]any{"status": "cancelled"}, nil
//	    }
//
//	    // Parse and use the configuration
//	    var config DeployConfig
//	    if err := result.DecodeContent(&config); err != nil {
//	        return nil, err
//	    }
//
//	    // Perform deployment with elicited config
//	    return performDeployment(config)
//	}
//
//	// Register the session-aware tool
//	handler, err := NewHandler(
//	    WithName("deploy-server"),
//	    WithSessionTool("deploy_app", "Deploy application with interactive configuration", deployApp),
//	)
func WithSessionTool[TIn, TOut any](name, description string, fn SessionAwareToolFunc[TIn, TOut]) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return ErrEmptyValue
		}

		// Create registration function that uses the session-aware handler
		registerFunc := func(server *mcp.Server) error {
			tool := &mcp.Tool{
				Name:        name,
				Description: description,
				// Let the generic AddTool handle schema generation
			}
			handler := createSessionAwareToolHandler(fn)
			mcp.AddTool(server, tool, handler)
			return nil
		}

		cfg.tools = append(cfg.tools, registerFunc)

		return nil
	}
}

// WithSessionPrompt adds a session-aware prompt with elicitation capability access.
// Session-aware prompts can elicit additional context or requirements from users
// to generate more targeted and personalized prompt responses.
//
// Example:
//
//	func dynamicDocPrompt(ctx context.Context, capability ElicitationCapability, args map[string]any) (*PromptResult, error) {
//	    docType := "document"
//	    if dt, ok := args["type"].(string); ok {
//	        docType = dt
//	    }
//
//	    // Elicit specific requirements for the document
//	    result, err := ElicitSimple(ctx, capability,
//	        fmt.Sprintf("What specific requirements do you have for your %s?", docType),
//	        "requirements", "Describe your specific needs and constraints")
//	    if err != nil {
//	        return nil, err
//	    }
//
//	    requirements := "No specific requirements provided"
//	    if result.IsAccepted() && result.GetContent() != nil {
//	        if req, ok := result.GetContent()["requirements"].(string); ok && req != "" {
//	            requirements = req
//	        }
//	    }
//
//	    return &PromptResult{
//	        Description: fmt.Sprintf("Interactive %s generation with custom requirements", docType),
//	        Messages: []PromptMessage{
//	            {Role: "system", Content: fmt.Sprintf("Create a %s that meets these requirements: %s", docType, requirements)},
//	            {Role: "user", Content: fmt.Sprintf("Generate the %s", docType)},
//	        },
//	    }, nil
//	}
//
//	// Register the session-aware prompt
//	handler, err := NewHandler(
//	    WithName("doc-server"),
//	    WithSessionPrompt("interactive_doc", "Generate documents with elicited requirements", dynamicDocPrompt),
//	)
func WithSessionPrompt(name, description string, fn SessionAwarePromptFunc) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return ErrEmptyValue
		}

		registerFunc := func(server *mcp.Server) error {
			prompt := &mcp.Prompt{
				Name:        name,
				Description: description,
			}
			handler := createSessionAwarePromptHandler(fn)
			server.AddPrompt(prompt, handler)
			return nil
		}

		cfg.prompts = append(cfg.prompts, registerFunc)
		return nil
	}
}

// WithSessionPromptWithArgs adds a session-aware prompt with argument definitions and elicitation capability access
func WithSessionPromptWithArgs(name, description string, args []*mcp.PromptArgument, fn SessionAwarePromptFunc) Option {
	return func(cfg *handlerConfig) error {
		if name == "" {
			return ErrEmptyValue
		}

		registerFunc := func(server *mcp.Server) error {
			prompt := &mcp.Prompt{
				Name:        name,
				Description: description,
				Arguments:   args,
			}
			handler := createSessionAwarePromptHandler(fn)
			server.AddPrompt(prompt, handler)
			return nil
		}

		cfg.prompts = append(cfg.prompts, registerFunc)
		return nil
	}
}

// WithSessionResource adds a session-aware resource with elicitation capability access.
// Session-aware resources can elicit authentication credentials, filter preferences,
// or other parameters needed to access or customize the resource content.
//
// Example:
//
//	func configResource(ctx context.Context, capability ElicitationCapability, uri string) (*ResourceContent, error) {
//	    // Elicit environment preference for configuration
//	    result, err := ElicitSimple(ctx, capability,
//	        "Which environment configuration would you like?",
//	        "environment", "Choose: development, staging, or production")
//	    if err != nil {
//	        return nil, err
//	    }
//
//	    env := "development" // default
//	    if result.IsAccepted() && result.GetContent() != nil {
//	        if e, ok := result.GetContent()["environment"].(string); ok {
//	            env = e
//	        }
//	    }
//
//	    // Load configuration for the specified environment
//	    config, err := loadConfigForEnvironment(env)
//	    if err != nil {
//	        return nil, err
//	    }
//
//	    return &ResourceContent{
//	        Content:  []byte(config),
//	        MIMEType: "application/json",
//	    }, nil
//	}
//
//	// Register the session-aware resource
//	handler, err := NewHandler(
//	    WithName("config-server"),
//	    WithSessionResource("config://app", "Application configuration with environment selection", configResource),
//	)
func WithSessionResource(uri, description string, fn SessionAwareResourceFunc) Option {
	return func(cfg *handlerConfig) error {
		if uri == "" {
			return ErrEmptyValue
		}

		registerFunc := func(server *mcp.Server) error {
			resource := &mcp.Resource{
				URI:         uri,
				Name:        uri, // Use URI as name by default
				Description: description,
			}
			handler := createSessionAwareResourceHandler(fn)
			server.AddResource(resource, handler)
			return nil
		}

		cfg.resources = append(cfg.resources, registerFunc)
		return nil
	}
}

// WithSessionResourceTemplate adds a session-aware resource template with elicitation capability access
func WithSessionResourceTemplate(uriTemplate, description string, fn SessionAwareResourceFunc) Option {
	return func(cfg *handlerConfig) error {
		if uriTemplate == "" {
			return ErrEmptyValue
		}

		registerFunc := func(server *mcp.Server) error {
			template := &mcp.ResourceTemplate{
				URITemplate: uriTemplate,
				Name:        uriTemplate, // Use template as name by default
				Description: description,
			}
			handler := createSessionAwareResourceHandler(fn)
			server.AddResourceTemplate(template, handler)
			return nil
		}

		cfg.resourceTemplates = append(cfg.resourceTemplates, registerFunc)
		return nil
	}
}
