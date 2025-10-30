package mcpio

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper types
type testInput struct {
	Value string `json:"value"`
}

type testOutput struct {
	Result string `json:"result"`
}

// Test helper functions
func testToolFunc(ctx context.Context, toolCtx ToolContext, input testInput) (testOutput, error) {
	return testOutput{Result: input.Value + "_processed"}, nil
}

func testRawToolFunc(ctx context.Context, toolCtx ToolContext, input []byte) ([]byte, error) {
	return input, nil
}

func TestBasicOptions(t *testing.T) {
	t.Parallel()

	validServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	t.Run("WithName", func(t *testing.T) {
		tests := []struct {
			name      string
			inputName string
			wantErr   error
		}{
			{
				name:      "valid name",
				inputName: "test-server",
				wantErr:   nil,
			},
			{
				name:      "empty name should return error",
				inputName: "",
				wantErr:   ErrEmptyValue,
			},
			{
				name:      "whitespace only name should be valid",
				inputName: "  ",
				wantErr:   nil,
			},
			{
				name:      "special characters in name",
				inputName: "test-server_v1.0",
				wantErr:   nil,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := &handlerConfig{}
				option := WithName(tt.inputName)
				err := option(cfg)

				if tt.wantErr != nil {
					require.ErrorIs(t, err, tt.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.inputName, cfg.name)
				}
			})
		}
	})

	t.Run("WithVersion", func(t *testing.T) {
		tests := []struct {
			name         string
			inputVersion string
			wantErr      error
		}{
			{
				name:         "valid version",
				inputVersion: "1.0.0",
				wantErr:      nil,
			},
			{
				name:         "empty version should return error",
				inputVersion: "",
				wantErr:      ErrEmptyValue,
			},
			{
				name:         "semantic version",
				inputVersion: "2.1.3-beta.1",
				wantErr:      nil,
			},
			{
				name:         "simple version string",
				inputVersion: "latest",
				wantErr:      nil,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := &handlerConfig{}
				option := WithVersion(tt.inputVersion)
				err := option(cfg)

				if tt.wantErr != nil {
					require.ErrorIs(t, err, tt.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.inputVersion, cfg.version)
				}
			})
		}
	})

	t.Run("WithServer", func(t *testing.T) {
		tests := []struct {
			name    string
			server  *mcp.Server
			wantErr error
		}{
			{
				name:    "valid server",
				server:  validServer,
				wantErr: nil,
			},
			{
				name:    "nil server should return error",
				server:  nil,
				wantErr: ErrNilValue,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := &handlerConfig{}
				option := WithServer(tt.server)
				err := option(cfg)

				if tt.wantErr != nil {
					require.ErrorIs(t, err, tt.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.server, cfg.server)
				}
			})
		}
	})
}

func TestMultipleOptionsApplication(t *testing.T) {
	t.Parallel()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	schema := &jsonschema.Schema{
		Type:        "object",
		Description: "Test schema",
		Properties: map[string]*jsonschema.Schema{
			"input": {
				Type:        "string",
				Description: "Test input",
			},
		},
		Required: []string{"input"},
	}

	cfg := &handlerConfig{
		tools: make([]toolRegisterFunc, 0),
	}
	options := []Option{
		WithName("multi-test"),
		WithVersion("2.0.0"),
		WithServer(server),
		WithTool("typed-tool", "A typed tool", testToolFunc),
		WithRawTool("raw-tool", "A raw tool", schema, testRawToolFunc),
	}

	// Apply all options
	for _, opt := range options {
		err := opt(cfg)
		require.NoError(t, err)
	}

	// Verify all config fields are set correctly
	assert.Equal(t, "multi-test", cfg.name)
	assert.Equal(t, "2.0.0", cfg.version)
	assert.Equal(t, server, cfg.server)
	assert.Len(t, cfg.tools, 2) // Two tools should be registered
}

func TestOptionErrorConditions(t *testing.T) {
	t.Parallel()

	// Test that errors from options don't affect config state
	cfg := &handlerConfig{}

	// Apply a valid option first
	err := WithName("test-server")(cfg)
	require.NoError(t, err)
	assert.Equal(t, "test-server", cfg.name)

	// Apply an invalid option - should not change existing state
	err = WithName("")(cfg)
	require.ErrorIs(t, err, ErrEmptyValue)
	assert.Equal(t, "test-server", cfg.name)

	// Apply another valid option
	err = WithVersion("1.0.0")(cfg)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", cfg.version)

	// Apply invalid version - should not change version state
	err = WithVersion("")(cfg)
	require.ErrorIs(t, err, ErrEmptyValue)
	assert.Equal(t, "1.0.0", cfg.version)
}

func TestWithServerOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    *mcp.ServerOptions
		wantErr error
	}{
		{
			name: "valid server options with all fields",
			opts: &mcp.ServerOptions{
				Instructions: "Test instructions",
				PageSize:     100,
				HasPrompts:   true,
				HasResources: true,
				HasTools:     true,
			},
			wantErr: nil,
		},
		{
			name:    "valid empty server options",
			opts:    &mcp.ServerOptions{},
			wantErr: nil,
		},
		{
			name:    "nil server options should return error",
			opts:    nil,
			wantErr: ErrNilValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{}
			option := WithServerOptions(tt.opts)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.opts, cfg.serverOptions)
			}
		})
	}
}

func TestWithStreamableHTTPOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    *mcp.StreamableHTTPOptions
		wantErr error
	}{
		{
			name: "stateless mode enabled",
			opts: &mcp.StreamableHTTPOptions{
				Stateless: true,
			},
			wantErr: nil,
		},
		{
			name: "JSON response mode enabled",
			opts: &mcp.StreamableHTTPOptions{
				JSONResponse: true,
			},
			wantErr: nil,
		},
		{
			name: "both options enabled",
			opts: &mcp.StreamableHTTPOptions{
				Stateless:    true,
				JSONResponse: true,
			},
			wantErr: nil,
		},
		{
			name:    "valid empty streamable HTTP options",
			opts:    &mcp.StreamableHTTPOptions{},
			wantErr: nil,
		},
		{
			name:    "nil streamable HTTP options should return error",
			opts:    nil,
			wantErr: ErrNilValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{}
			option := WithStreamableHTTPOptions(tt.opts)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.opts, cfg.streamableHTTPOptions)
			}
		})
	}
}

func TestPromptOptions(t *testing.T) {
	t.Parallel()

	testPromptFunc := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		return &PromptResult{
			Messages: []PromptMessage{{Role: "user", Content: "test"}},
		}, nil
	}

	t.Run("WithPrompt", func(t *testing.T) {
		tests := []struct {
			name          string
			promptName    string
			description   string
			promptFunc    PromptFunc
			wantErr       error
			expectPrompts int
		}{
			{
				name:          "valid prompt",
				promptName:    "test-prompt",
				description:   "A test prompt",
				promptFunc:    testPromptFunc,
				wantErr:       nil,
				expectPrompts: 1,
			},
			{
				name:          "empty prompt name should return error",
				promptName:    "",
				description:   "A test prompt",
				promptFunc:    testPromptFunc,
				wantErr:       ErrEmptyValue,
				expectPrompts: 0,
			},
			{
				name:          "empty description should be valid",
				promptName:    "test-prompt",
				description:   "",
				promptFunc:    testPromptFunc,
				wantErr:       nil,
				expectPrompts: 1,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := &handlerConfig{
					prompts: make([]promptRegisterFunc, 0),
				}
				option := WithPrompt(tt.promptName, tt.description, tt.promptFunc)
				err := option(cfg)

				if tt.wantErr != nil {
					require.ErrorIs(t, err, tt.wantErr)
				} else {
					require.NoError(t, err)
				}
				assert.Len(t, cfg.prompts, tt.expectPrompts)
			})
		}
	})

	t.Run("WithPromptWithArgs", func(t *testing.T) {
		promptArgs := []*mcp.PromptArgument{
			{
				Name:        "name",
				Description: "The name to use",
				Required:    true,
			},
		}

		tests := []struct {
			name          string
			promptName    string
			description   string
			args          []*mcp.PromptArgument
			promptFunc    PromptFunc
			wantErr       error
			expectPrompts int
		}{
			{
				name:          "valid prompt with args",
				promptName:    "test-prompt",
				description:   "A test prompt",
				args:          promptArgs,
				promptFunc:    testPromptFunc,
				wantErr:       nil,
				expectPrompts: 1,
			},
			{
				name:          "empty prompt name should return error",
				promptName:    "",
				description:   "A test prompt",
				args:          promptArgs,
				promptFunc:    testPromptFunc,
				wantErr:       ErrEmptyValue,
				expectPrompts: 0,
			},
			{
				name:          "nil args should be valid",
				promptName:    "test-prompt",
				description:   "A test prompt",
				args:          nil,
				promptFunc:    testPromptFunc,
				wantErr:       nil,
				expectPrompts: 1,
			},
			{
				name:          "empty args slice should be valid",
				promptName:    "test-prompt",
				description:   "A test prompt",
				args:          []*mcp.PromptArgument{},
				promptFunc:    testPromptFunc,
				wantErr:       nil,
				expectPrompts: 1,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := &handlerConfig{
					prompts: make([]promptRegisterFunc, 0),
				}
				option := WithPromptWithArgs(tt.promptName, tt.description, tt.args, tt.promptFunc)
				err := option(cfg)

				if tt.wantErr != nil {
					require.ErrorIs(t, err, tt.wantErr)
				} else {
					require.NoError(t, err)
				}
				assert.Len(t, cfg.prompts, tt.expectPrompts)
			})
		}

		// Test that the registration closure works correctly
		t.Run("registration function executes", func(t *testing.T) {
			handler, err := NewHandler(
				WithName("test-server"),
				WithPromptWithArgs("greeting", "Generate greeting with args", promptArgs, testPromptFunc),
			)
			require.NoError(t, err)
			assert.NotNil(t, handler)
			assert.NotNil(t, handler.GetServer())
		})
	})
}

func TestResourceOptions(t *testing.T) {
	t.Parallel()

	resourceFunc := func(ctx context.Context, uri string) (*ResourceContent, error) {
		return &ResourceContent{Content: []byte("test content"), MIMEType: "text/plain"}, nil
	}

	templateFunc := func(ctx context.Context, uri string) (*ResourceContent, error) {
		return &ResourceContent{Content: []byte("template content"), MIMEType: "application/json"}, nil
	}

	t.Run("WithResource", func(t *testing.T) {
		// Test valid resource
		cfg := &handlerConfig{resources: make([]resourceRegisterFunc, 0)}
		err := WithResource("test://resource", "A test resource", resourceFunc)(cfg)
		require.NoError(t, err)
		assert.Len(t, cfg.resources, 1)

		// Test empty URI error
		err = WithResource("", "A test resource", resourceFunc)(&handlerConfig{})
		require.ErrorIs(t, err, ErrEmptyValue)

		// Test empty description valid
		cfg = &handlerConfig{resources: make([]resourceRegisterFunc, 0)}
		err = WithResource("test://resource", "", resourceFunc)(cfg)
		require.NoError(t, err)
		assert.Len(t, cfg.resources, 1)

		// Test complex URI valid
		cfg = &handlerConfig{resources: make([]resourceRegisterFunc, 0)}
		err = WithResource("file:///path/to/resource.txt", "A file resource", resourceFunc)(cfg)
		require.NoError(t, err)
		assert.Len(t, cfg.resources, 1)
	})

	t.Run("WithResourceTemplate", func(t *testing.T) {
		// Valid template test
		cfg := &handlerConfig{resourceTemplates: make([]resourceTemplateRegisterFunc, 0)}
		err := WithResourceTemplate("user://{id}", "A user template", templateFunc)(cfg)
		require.NoError(t, err)
		assert.Len(t, cfg.resourceTemplates, 1)

		// Empty template error test
		err = WithResourceTemplate("", "A test template", templateFunc)(&handlerConfig{})
		require.ErrorIs(t, err, ErrEmptyValue)

		// Empty description valid test
		cfg = &handlerConfig{resourceTemplates: make([]resourceTemplateRegisterFunc, 0)}
		err = WithResourceTemplate("config://{section}", "", templateFunc)(cfg)
		require.NoError(t, err)
		assert.Len(t, cfg.resourceTemplates, 1)

		// Multiple placeholders valid test with more detailed checks
		cfg = &handlerConfig{resourceTemplates: make([]resourceTemplateRegisterFunc, 0)}
		complexTemplate := "api://v1/users/{userId}/posts/{postId}"
		err = WithResourceTemplate(complexTemplate, "Complex template", templateFunc)(cfg)
		require.NoError(t, err)
		assert.Len(t, cfg.resourceTemplates, 1)
	})
}

// Test types for tool examples
type GreetInput struct {
	Name string `json:"name" jsonschema:"Name to greet"`
}

type GreetOutput struct {
	Message string `json:"message" jsonschema:"Greeting message"`
}

// Test helper functions for tool tests
func greetFunc(ctx context.Context, toolCtx ToolContext, input GreetInput) (GreetOutput, error) {
	return GreetOutput{Message: "Hello, " + input.Name}, nil
}

func farewellFunc(ctx context.Context, toolCtx ToolContext, input GreetInput) (GreetOutput, error) {
	return GreetOutput{Message: "Goodbye, " + input.Name}, nil
}

func rawToolFunc(ctx context.Context, toolCtx ToolContext, input []byte) ([]byte, error) {
	return []byte(`{"result": "processed"}`), nil
}

func TestWithTypedTool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		toolName    string
		description string
		wantErr     error
	}{
		{
			name:        "valid tool",
			toolName:    "greet",
			description: "Greet someone by name",
			wantErr:     nil,
		},
		{
			name:        "empty tool name error",
			toolName:    "",
			description: "description",
			wantErr:     ErrEmptyValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewHandler(WithTool(tt.toolName, tt.description, greetFunc))

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWithRawTool(t *testing.T) {
	t.Parallel()
	s := &jsonschema.Schema{
		Type:        "object",
		Description: "Raw tool input",
		Properties: map[string]*jsonschema.Schema{
			"data": {
				Type:        "string",
				Description: "Input data",
			},
		},
		Required: []string{"data"},
	}

	tests := []struct {
		name        string
		toolName    string
		description string
		s           interface{}
		wantErr     error
	}{
		{
			name:        "valid raw tool",
			toolName:    "process",
			description: "Process raw data",
			s:           s,
			wantErr:     nil,
		},
		{
			name:        "empty tool name error",
			toolName:    "",
			description: "description",
			s:           s,
			wantErr:     ErrEmptyValue,
		},
		{
			name:        "nil schema error",
			toolName:    "test",
			description: "description",
			s:           nil,
			wantErr:     ErrNilValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var schemaPtr *jsonschema.Schema
			if tt.s != nil {
				schemaPtr = tt.s.(*jsonschema.Schema)
			}

			_, err := NewHandler(WithRawTool(tt.toolName, tt.description, schemaPtr, rawToolFunc))

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Helper function to create test requests for raw tool handlers
func createTestCallToolRequest(t *testing.T, args map[string]any) *mcp.CallToolRequest {
	t.Helper()
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal test arguments: %v", err)
	}
	return &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: json.RawMessage(argsJSON),
		},
	}
}

// Test createRawToolHandler function directly
func TestCreateRawToolHandler(t *testing.T) {
	t.Parallel()

	t.Run("success path with valid JSON", func(t *testing.T) {
		// Raw function that returns valid JSON
		rawFunc := func(ctx context.Context, toolCtx ToolContext, input []byte) ([]byte, error) {
			return []byte(`{"result": "success", "input_length": ` + strconv.Itoa(len(input)) + `}`), nil
		}

		h := &Handler{contextStore: NewContextStore(DefaultContextKey)}
		handler := createRawToolHandler(h, rawFunc)

		req := createTestCallToolRequest(t, map[string]any{
			"test": "value",
			"num":  42,
		})

		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		assert.Contains(t, textContent.Text, `"result": "success"`)
	})

	t.Run("input marshaling error", func(t *testing.T) {
		// Raw function (won't be called due to marshaling error)
		rawFunc := func(ctx context.Context, toolCtx ToolContext, input []byte) ([]byte, error) {
			return []byte(`{"result": "should not reach"}`), nil
		}

		h := &Handler{contextStore: NewContextStore(DefaultContextKey)}
		handler := createRawToolHandler(h, rawFunc)

		// Create request with invalid JSON that will cause marshaling to fail
		// We'll pass invalid JSON as RawMessage directly
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Arguments: json.RawMessage(`{invalid json content`), // Invalid JSON
			},
		}

		result, err := handler(context.Background(), req)

		require.NoError(t, err) // Handler should not return protocol error
		assert.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		assert.Contains(t, textContent.Text, "Failed to marshal input")
	})

	t.Run("raw function returns ToolError", func(t *testing.T) {
		// Raw function that returns a ToolError
		rawFunc := func(ctx context.Context, toolCtx ToolContext, input []byte) ([]byte, error) {
			return nil, NewToolError("validation failed")
		}

		h := &Handler{contextStore: NewContextStore(DefaultContextKey)}
		handler := createRawToolHandler(h, rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"test": "data"})

		result, err := handler(context.Background(), req)

		require.NoError(t, err) // Handler should not return protocol error
		assert.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		assert.Equal(t, "validation failed", textContent.Text)
	})

	t.Run("raw function returns ToolError with code", func(t *testing.T) {
		// Raw function that returns a ToolError with error code
		rawFunc := func(ctx context.Context, toolCtx ToolContext, input []byte) ([]byte, error) {
			return nil, ValidationError("invalid input format")
		}

		h := &Handler{contextStore: NewContextStore(DefaultContextKey)}
		handler := createRawToolHandler(h, rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"data": "invalid"})

		result, err := handler(context.Background(), req)

		require.NoError(t, err) // Handler should not return protocol error
		assert.NotNil(t, result)
		assert.True(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		// Error codes should be included with [CODE] prefix
		assert.Equal(t, "[VALIDATION_ERROR] invalid input format", textContent.Text)
	})

	t.Run("raw function returns protocol error", func(t *testing.T) {
		errDatabaseFailed := errors.New("database connection failed")

		// Raw function that returns a regular Go error (protocol error)
		rawFunc := func(ctx context.Context, toolCtx ToolContext, input []byte) ([]byte, error) {
			return nil, errDatabaseFailed
		}

		h := &Handler{contextStore: NewContextStore(DefaultContextKey)}
		handler := createRawToolHandler(h, rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"query": "test"})

		result, err := handler(context.Background(), req)

		// Protocol errors should be returned as Go errors, not CallToolResult
		require.ErrorIs(t, err, errDatabaseFailed)
		assert.Nil(t, result)
	})

	t.Run("invalid JSON output", func(t *testing.T) {
		// Raw function that returns invalid JSON
		rawFunc := func(ctx context.Context, toolCtx ToolContext, input []byte) ([]byte, error) {
			return []byte(`{invalid json:`), nil
		}

		h := &Handler{contextStore: NewContextStore(DefaultContextKey)}
		handler := createRawToolHandler(h, rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"test": "data"})

		result, err := handler(context.Background(), req)

		// Invalid JSON should be a protocol error
		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrInvalidJSON)
	})

	t.Run("empty JSON object output", func(t *testing.T) {
		// Raw function that returns empty JSON object
		rawFunc := func(ctx context.Context, toolCtx ToolContext, input []byte) ([]byte, error) {
			return []byte(`{}`), nil
		}

		h := &Handler{contextStore: NewContextStore(DefaultContextKey)}
		handler := createRawToolHandler(h, rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"test": "data"})

		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		assert.Equal(t, "{}", textContent.Text)
	})

	t.Run("JSON array output", func(t *testing.T) {
		// Raw function that returns JSON array
		rawFunc := func(ctx context.Context, toolCtx ToolContext, input []byte) ([]byte, error) {
			return []byte(`[1, 2, 3]`), nil
		}

		h := &Handler{contextStore: NewContextStore(DefaultContextKey)}
		handler := createRawToolHandler(h, rawFunc)

		req := createTestCallToolRequest(t, map[string]any{"count": 3})

		result, err := handler(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		assert.True(t, ok)
		assert.Equal(t, "[1, 2, 3]", textContent.Text)
	})
}
