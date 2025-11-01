package mcpio

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for createTypedHandler function from tool.go
// Helper types (SimpleInput, SimpleOutput, simpleEchoFunc) are shared from handler_test.go

func TestCreateTypedHandlerSuccess(t *testing.T) {
	t.Parallel()
	h := &Handler{}
	handlerFunc := createTypedHandler(h, simpleEchoFunc)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: "test_tool"},
	}

	input := SimpleInput{Text: "hello world"}
	result, output, err := handlerFunc(t.Context(), req, input)

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "hello world", output.Message)
}

func TestCreateTypedHandlerToolError(t *testing.T) {
	t.Parallel()
	// Function that returns a tool error
	errorFunc := func(ctx context.Context, toolCtx RequestContext, input SimpleInput) (SimpleOutput, error) {
		return SimpleOutput{}, NewToolError("tool failed")
	}

	h := &Handler{}
	handlerFunc := createTypedHandler(h, errorFunc)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: "test_tool"},
	}

	input := SimpleInput{Text: "test"}
	result, output, err := handlerFunc(t.Context(), req, input)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, SimpleOutput{}, output)

	var toolErr *ToolError
	require.ErrorAs(t, err, &toolErr)
	assert.Equal(t, "tool failed", toolErr.Message)
}

func TestCreateTypedHandlerProtocolError(t *testing.T) {
	t.Parallel()
	// Function that returns a non-tool error
	errorFunc := func(ctx context.Context, toolCtx RequestContext, input SimpleInput) (SimpleOutput, error) {
		return SimpleOutput{}, errors.New("protocol error")
	}

	h := &Handler{}
	handlerFunc := createTypedHandler(h, errorFunc)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: "test_tool"},
	}

	input := SimpleInput{Text: "test"}
	result, output, err := handlerFunc(t.Context(), req, input)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, SimpleOutput{}, output)
	assert.Equal(t, "protocol error", err.Error())
}
