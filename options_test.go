package mcpio

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test input/output types for testing
type testInput struct {
	Value string `json:"value"`
}

type testOutput struct {
	Result string `json:"result"`
}

// Test tool function
func testToolFunc(ctx context.Context, input testInput) (testOutput, error) {
	return testOutput{Result: "processed: " + input.Value}, nil
}

// Test raw tool function
func testRawToolFunc(ctx context.Context, input []byte) ([]byte, error) {
	return []byte(`{"processed": true}`), nil
}

func TestWithName(t *testing.T) {
	t.Parallel()

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
			wantErr:   ErrEmptyName,
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
}

func TestWithVersion(t *testing.T) {
	t.Parallel()

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
			wantErr:      ErrEmptyVersion,
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
}

func TestWithTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		toolName    string
		description string
		toolFunc    ToolFunc[testInput, testOutput]
		wantErr     error
	}{
		{
			name:        "valid tool",
			toolName:    "test-tool",
			description: "A test tool",
			toolFunc:    testToolFunc,
			wantErr:     nil,
		},
		{
			name:        "empty tool name should return error",
			toolName:    "",
			description: "A test tool",
			toolFunc:    testToolFunc,
			wantErr:     ErrEmptyToolName,
		},
		{
			name:        "empty description should be valid",
			toolName:    "test-tool",
			description: "",
			toolFunc:    testToolFunc,
			wantErr:     nil,
		},
		{
			name:        "tool with special characters in name",
			toolName:    "test_tool-v1",
			description: "A test tool with special chars",
			toolFunc:    testToolFunc,
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{
				tools: make([]toolRegisterFunc, 0),
			}
			option := WithTool(tt.toolName, tt.description, tt.toolFunc)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				// Should not add tool registration func on error
				assert.Empty(t, cfg.tools)
			} else {
				require.NoError(t, err)
				// Should add exactly one tool registration function
				assert.Len(t, cfg.tools, 1)
			}
		})
	}
}

func TestWithRawToolOptions(t *testing.T) {
	t.Parallel()

	validSchema := schema.NewObject(
		"Test input schema",
		map[string]string{"value": "Test value"},
		[]string{"value"},
	)

	// Example function matching the documentation
	rawFunc := func(ctx context.Context, input []byte) ([]byte, error) { return input, nil }

	tests := []struct {
		name        string
		toolName    string
		description string
		schema      any
		toolFunc    RawToolFunc
		wantErr     error
	}{
		{
			name:        "valid raw tool",
			toolName:    "raw-tool",
			description: "A raw test tool",
			schema:      validSchema,
			toolFunc:    testRawToolFunc,
			wantErr:     nil,
		},
		{
			name:        "WithRawTool with JSON string schema",
			toolName:    "process",
			description: "Process JSON",
			schema:      `{"type":"object"}`,
			toolFunc:    rawFunc,
			wantErr:     nil,
		},
		{
			name:        "empty tool name should return error",
			toolName:    "",
			description: "A raw test tool",
			schema:      validSchema,
			toolFunc:    testRawToolFunc,
			wantErr:     ErrEmptyToolName,
		},
		{
			name:        "nil schema should return error",
			toolName:    "raw-tool",
			description: "A raw test tool",
			schema:      nil,
			toolFunc:    testRawToolFunc,
			wantErr:     ErrNilSchema,
		},
		{
			name:        "empty description should be valid",
			toolName:    "raw-tool",
			description: "",
			schema:      validSchema,
			toolFunc:    testRawToolFunc,
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &handlerConfig{
				tools: make([]toolRegisterFunc, 0),
			}
			option := WithRawTool(tt.toolName, tt.description, tt.schema, tt.toolFunc)
			err := option(cfg)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				// Should not add tool registration func on error
				assert.Empty(t, cfg.tools)
			} else {
				require.NoError(t, err)
				// Should add exactly one tool registration function
				assert.Len(t, cfg.tools, 1)
			}
		})
	}
}

func TestWithServer(t *testing.T) {
	t.Parallel()

	validServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

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
			wantErr: ErrNilServer,
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
}

func TestMultipleOptionsApplication(t *testing.T) {
	t.Parallel()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	schema := schema.NewObject(
		"Test schema",
		map[string]string{"input": "Test input"},
		[]string{"input"},
	)

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

func TestWithPrompt(t *testing.T) {
	t.Parallel()

	testPromptFunc := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		return &PromptResult{
			Messages: []PromptMessage{{Role: "user", Content: "test"}},
		}, nil
	}

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
			wantErr:       ErrEmptyPromptName,
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
}

func TestWithPromptWithArgs(t *testing.T) {
	t.Parallel()

	testPromptFunc := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		return &PromptResult{
			Messages: []PromptMessage{{Role: "user", Content: "test"}},
		}, nil
	}

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
			wantErr:       ErrEmptyPromptName,
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
}

func TestWithResource(t *testing.T) {
	t.Parallel()

	resourceFunc := func(ctx context.Context, uri string) (*ResourceContent, error) {
		return &ResourceContent{Content: []byte("test content"), MIMEType: "text/plain"}, nil
	}

	// Test valid resource
	cfg := &handlerConfig{resources: make([]resourceRegisterFunc, 0)}
	err := WithResource("test://resource", "A test resource", resourceFunc)(cfg)
	require.NoError(t, err)
	assert.Len(t, cfg.resources, 1)

	// Test empty URI error
	err = WithResource("", "A test resource", resourceFunc)(&handlerConfig{})
	require.ErrorIs(t, err, ErrEmptyResourceURI)

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
}

func TestWithResourceTemplate(t *testing.T) {
	t.Parallel()

	templateFunc := func(ctx context.Context, uri string) (*ResourceContent, error) {
		return &ResourceContent{Content: []byte("template content"), MIMEType: "application/json"}, nil
	}

	// Valid template test
	cfg := &handlerConfig{resourceTemplates: make([]resourceTemplateRegisterFunc, 0)}
	err := WithResourceTemplate("user://{id}", "A user template", templateFunc)(cfg)
	require.NoError(t, err)
	assert.Len(t, cfg.resourceTemplates, 1)

	// Empty template error test
	err = WithResourceTemplate("", "A test template", templateFunc)(&handlerConfig{})
	require.ErrorIs(t, err, ErrEmptyResourceTemplate)

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
	require.ErrorIs(t, err, ErrEmptyName)
	assert.Equal(t, "test-server", cfg.name)

	// Apply another valid option
	err = WithVersion("1.0.0")(cfg)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", cfg.version)

	// Apply invalid version - should not change version state
	err = WithVersion("")(cfg)
	require.ErrorIs(t, err, ErrEmptyVersion)
	assert.Equal(t, "1.0.0", cfg.version)
}
