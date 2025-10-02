package mcpio

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
