package mcpio

import (
	"context"
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
