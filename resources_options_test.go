package mcpio

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
}
