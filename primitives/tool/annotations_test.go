package tool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolAnnotations_toMCP(t *testing.T) {
	t.Parallel()

	t.Run("nil annotations", func(t *testing.T) {
		var a *ToolAnnotations
		result := a.toMCP()
		assert.Nil(t, result)
	})

	t.Run("empty annotations", func(t *testing.T) {
		a := &ToolAnnotations{}
		result := a.toMCP()

		require.NotNil(t, result)
		assert.False(t, result.ReadOnlyHint)
		assert.False(t, result.IdempotentHint)
		assert.Nil(t, result.DestructiveHint)
		assert.Nil(t, result.OpenWorldHint)
	})

	t.Run("boolean hints", func(t *testing.T) {
		a := &ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
		}
		result := a.toMCP()

		require.NotNil(t, result)
		assert.True(t, result.ReadOnlyHint)
		assert.True(t, result.IdempotentHint)
		assert.Nil(t, result.DestructiveHint)
		assert.Nil(t, result.OpenWorldHint)
	})

	t.Run("pointer hints - destructive true", func(t *testing.T) {
		destructive := true
		a := &ToolAnnotations{
			DestructiveHint: &destructive,
		}
		result := a.toMCP()

		require.NotNil(t, result)
		require.NotNil(t, result.DestructiveHint)
		assert.True(t, *result.DestructiveHint)
	})

	t.Run("pointer hints - destructive false", func(t *testing.T) {
		destructive := false
		a := &ToolAnnotations{
			DestructiveHint: &destructive,
		}
		result := a.toMCP()

		require.NotNil(t, result)
		require.NotNil(t, result.DestructiveHint)
		assert.False(t, *result.DestructiveHint)
	})

	t.Run("pointer hints - open world true", func(t *testing.T) {
		openWorld := true
		a := &ToolAnnotations{
			OpenWorldHint: &openWorld,
		}
		result := a.toMCP()

		require.NotNil(t, result)
		require.NotNil(t, result.OpenWorldHint)
		assert.True(t, *result.OpenWorldHint)
	})

	t.Run("pointer hints - open world false", func(t *testing.T) {
		openWorld := false
		a := &ToolAnnotations{
			OpenWorldHint: &openWorld,
		}
		result := a.toMCP()

		require.NotNil(t, result)
		require.NotNil(t, result.OpenWorldHint)
		assert.False(t, *result.OpenWorldHint)
	})

	t.Run("all fields set", func(t *testing.T) {
		destructive := true
		openWorld := false
		a := &ToolAnnotations{
			ReadOnlyHint:    true,
			IdempotentHint:  false,
			DestructiveHint: &destructive,
			OpenWorldHint:   &openWorld,
		}
		result := a.toMCP()

		require.NotNil(t, result)
		assert.True(t, result.ReadOnlyHint)
		assert.False(t, result.IdempotentHint)
		require.NotNil(t, result.DestructiveHint)
		assert.True(t, *result.DestructiveHint)
		require.NotNil(t, result.OpenWorldHint)
		assert.False(t, *result.OpenWorldHint)
	})
}
