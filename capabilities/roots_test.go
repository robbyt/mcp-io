package capabilities_test

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSupportsRoots(t *testing.T) {
	t.Parallel()

	t.Run("True", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		caps := &mcp.ClientCapabilities{}
		caps.Roots.ListChanged = true
		mockSession.On("InitializeParams").Return(&mcp.InitializeParams{
			Capabilities: caps,
		})

		session := capabilities.NewSession(mockSession.MockServerSession)
		supported := session.SupportsRoots()

		assert.True(t, supported)
		mockSession.AssertExpectations(t)
	})

	t.Run("False", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		caps := &mcp.ClientCapabilities{}
		caps.Roots.ListChanged = false
		mockSession.On("InitializeParams").Return(&mcp.InitializeParams{
			Capabilities: caps,
		})

		session := capabilities.NewSession(mockSession.MockServerSession)
		supported := session.SupportsRoots()

		assert.False(t, supported)
		mockSession.AssertExpectations(t)
	})
}

func TestListRoots(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.ListRootsResult{
			Roots: []*mcp.Root{
				{URI: "file:///project", Name: "project"},
				{URI: "file:///src", Name: "src"},
			},
		}
		mockSession.On("ListRoots", mock.Anything, mock.Anything).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		roots, err := session.ListRoots(context.Background())

		require.NoError(t, err)
		require.Len(t, roots, 2)
		assert.Equal(t, "file:///project", roots[0].URI)
		assert.Equal(t, "project", roots[0].Name)
		assert.Equal(t, "file:///src", roots[1].URI)
		assert.Equal(t, "src", roots[1].Name)
		mockSession.AssertExpectations(t)
	})

	t.Run("Success_EmptyList", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedResult := &mcp.ListRootsResult{
			Roots: []*mcp.Root{},
		}
		mockSession.On("ListRoots", mock.Anything, mock.Anything).Return(expectedResult, nil)

		session := capabilities.NewSession(mockSession.MockServerSession)
		roots, err := session.ListRoots(context.Background())

		require.NoError(t, err)
		assert.Empty(t, roots)
		mockSession.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockSession := testutil.NewMockSession()
		expectedErr := assert.AnError
		mockSession.On("ListRoots", mock.Anything, mock.Anything).Return(nil, expectedErr)

		session := capabilities.NewSession(mockSession.MockServerSession)
		roots, err := session.ListRoots(context.Background())

		require.Error(t, err)
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, roots)
		mockSession.AssertExpectations(t)
	})
}
