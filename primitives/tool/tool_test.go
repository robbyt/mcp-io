package tool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tool, err := New("test_tool", "Test description", `{"type":"object"}`)

	require.NoError(t, err)
	require.NotNil(t, tool)
	assert.Equal(t, "test_tool", tool.Name)
	assert.Equal(t, "Test description", tool.Description)
	assert.NotNil(t, tool.InputSchema)
}

func TestNew_Required(t *testing.T) {
	t.Run("valid tool", func(t *testing.T) {
		tool, err := New("my_tool", "My description", `{"type":"object"}`)

		require.NoError(t, err)
		require.NotNil(t, tool)
		assert.Equal(t, "my_tool", tool.Name)
		assert.Equal(t, "My description", tool.Description)
		assert.NotNil(t, tool.InputSchema)
	})

	t.Run("empty name", func(t *testing.T) {
		tool, err := New("", "Description", `{"type":"object"}`)

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("empty description", func(t *testing.T) {
		tool, err := New("tool", "", `{"type":"object"}`)

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "description cannot be empty")
	})

	t.Run("nil input schema", func(t *testing.T) {
		tool, err := New("tool", "Description", nil)

		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "input schema cannot be nil")
	})
}

func TestWithTitle(t *testing.T) {
	tool, err := New("tool", "Description", `{"type":"object"}`, WithTitle("My Tool"))

	require.NoError(t, err)
	assert.Equal(t, "My Tool", tool.Title)
}

func TestWithOutputSchema(t *testing.T) {
	tool, err := New("tool", "Description", `{"type":"object"}`,
		WithOutputSchema(`{"type":"object","properties":{"result":{"type":"string"}}}`))

	require.NoError(t, err)
	assert.NotNil(t, tool.OutputSchema)
}

func TestWithAnnotations(t *testing.T) {
	tool, err := New("tool", "Description", `{"type":"object"}`,
		WithAnnotations(&ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
		}))

	require.NoError(t, err)
	require.NotNil(t, tool.Annotations)
	assert.True(t, tool.Annotations.ReadOnlyHint)
	assert.True(t, tool.Annotations.IdempotentHint)
}

func TestWithMeta(t *testing.T) {
	tool, err := New("tool", "Description", `{"type":"object"}`,
		WithMeta(map[string]any{
			"version": "1.0.0",
			"author":  "test-team",
		}))

	require.NoError(t, err)
	require.NotNil(t, tool.Meta)
	assert.Equal(t, "1.0.0", tool.Meta["version"])
	assert.Equal(t, "test-team", tool.Meta["author"])
}

func TestWithReadOnly(t *testing.T) {
	tool, err := New("tool", "Description", `{"type":"object"}`, WithReadOnly())

	require.NoError(t, err)
	require.NotNil(t, tool.Annotations)
	assert.True(t, tool.Annotations.ReadOnlyHint)
}

func TestWithIdempotent(t *testing.T) {
	tool, err := New("tool", "Description", `{"type":"object"}`, WithIdempotent())

	require.NoError(t, err)
	require.NotNil(t, tool.Annotations)
	assert.True(t, tool.Annotations.IdempotentHint)
}

func TestWithDestructive(t *testing.T) {
	tool, err := New("tool", "Description", `{"type":"object"}`, WithDestructive())

	require.NoError(t, err)
	require.NotNil(t, tool.Annotations)
	require.NotNil(t, tool.Annotations.DestructiveHint)
	assert.True(t, *tool.Annotations.DestructiveHint)
}

func TestWithOpenWorld(t *testing.T) {
	tool, err := New("tool", "Description", `{"type":"object"}`, WithOpenWorld())

	require.NoError(t, err)
	require.NotNil(t, tool.Annotations)
	require.NotNil(t, tool.Annotations.OpenWorldHint)
	assert.True(t, *tool.Annotations.OpenWorldHint)
}

func TestWithClosedWorld(t *testing.T) {
	tool, err := New("tool", "Description", `{"type":"object"}`, WithClosedWorld())

	require.NoError(t, err)
	require.NotNil(t, tool.Annotations)
	require.NotNil(t, tool.Annotations.OpenWorldHint)
	assert.False(t, *tool.Annotations.OpenWorldHint)
}

func TestOptionChaining(t *testing.T) {
	tool, err := New("my_tool", "My tool", `{"type":"object"}`,
		WithTitle("My Awesome Tool"),
		WithReadOnly(),
		WithIdempotent(),
		WithMeta(map[string]any{"version": "2.0"}),
	)

	require.NoError(t, err)
	assert.Equal(t, "My Awesome Tool", tool.Title)
	require.NotNil(t, tool.Annotations)
	assert.True(t, tool.Annotations.ReadOnlyHint)
	assert.True(t, tool.Annotations.IdempotentHint)
	require.NotNil(t, tool.Meta)
	assert.Equal(t, "2.0", tool.Meta["version"])
}

func TestMultipleAnnotationOptions(t *testing.T) {
	tool, err := New("tool", "Description", `{"type":"object"}`,
		WithReadOnly(),
		WithIdempotent(),
		WithDestructive(),
		WithOpenWorld(),
	)

	require.NoError(t, err)
	require.NotNil(t, tool.Annotations)
	assert.True(t, tool.Annotations.ReadOnlyHint)
	assert.True(t, tool.Annotations.IdempotentHint)
	require.NotNil(t, tool.Annotations.DestructiveHint)
	assert.True(t, *tool.Annotations.DestructiveHint)
	require.NotNil(t, tool.Annotations.OpenWorldHint)
	assert.True(t, *tool.Annotations.OpenWorldHint)
}
