package mcpio

import (
	"context"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
			cfg := &handlerConfig{}
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

	validSchema := CreateObjectSchema(
		"Test input schema",
		map[string]string{"value": "Test value"},
		[]string{"value"},
	)

	tests := []struct {
		name        string
		toolName    string
		description string
		schema      *jsonschema.Schema
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
			cfg := &handlerConfig{}
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

	schema := CreateObjectSchema(
		"Test schema",
		map[string]string{"input": "Test input"},
		[]string{"input"},
	)

	cfg := &handlerConfig{}
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
	require.ErrorIs(t, err, ErrEmptyName)
	assert.Equal(t, "test-server", cfg.name) // Should remain unchanged

	// Apply another valid option
	err = WithVersion("1.0.0")(cfg)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", cfg.version)

	// Apply invalid version - should not change version state
	err = WithVersion("")(cfg)
	require.ErrorIs(t, err, ErrEmptyVersion)
	assert.Equal(t, "1.0.0", cfg.version) // Should remain unchanged
}
