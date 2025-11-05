//go:build integration

package mcpio_test

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type CompletionIntegrationSuite struct {
	testutil.IntegrationSuite
}

func TestCompletionIntegration(t *testing.T) {
	suite.Run(t, new(CompletionIntegrationSuite))
}

func (s *CompletionIntegrationSuite) TestBasicCompletion() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("completion-test"),
		mcpio.WithCompletion(func(ctx context.Context, reqCtx mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
			if ref.Type == "ref/prompt" {
				return &mcpio.CompletionResult{
					Values: []string{"value1", "value2", "value3"},
				}, nil
			}
			return nil, mcpio.NewCompletionError("unsupported type")
		}),
	)
	s.Require().NoError(err)

	client := testutil.ConnectInMemory(s.T(), handler)

	// Send completion request
	result, err := client.Complete(context.Background(), &mcp.CompleteParams{
		Ref: &mcp.CompleteReference{
			Type: "ref/prompt",
		},
	})

	s.Require().NoError(err)
	s.Len(result.Completion.Values, 3)
	s.Contains(result.Completion.Values, "value1")
	s.False(result.Completion.HasMore)
}

func (s *CompletionIntegrationSuite) TestCompletionWithPagination() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("completion-test"),
		mcpio.WithCompletion(func(ctx context.Context, reqCtx mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
			return &mcpio.CompletionResult{
				Values:  []string{"val1", "val2"},
				HasMore: true,
				Total:   100,
			}, nil
		}),
	)
	s.Require().NoError(err)

	client := testutil.ConnectInMemory(s.T(), handler)

	result, err := client.Complete(context.Background(), &mcp.CompleteParams{
		Ref: &mcp.CompleteReference{Type: "ref/resource"},
	})

	s.Require().NoError(err)
	s.Len(result.Completion.Values, 2)
	s.True(result.Completion.HasMore)
	s.Equal(100, result.Completion.Total)
}

func (s *CompletionIntegrationSuite) TestCompletionWithPromptArgument() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("completion-test"),
		mcpio.WithCompletion(func(ctx context.Context, reqCtx mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
			if ref.Type == "ref/prompt" && ref.Name == "greet" && ref.Argument == "language" {
				return &mcpio.CompletionResult{
					Values: []string{"English", "Spanish", "French"},
				}, nil
			}
			return &mcpio.CompletionResult{
				Values: []string{},
			}, nil
		}),
	)
	s.Require().NoError(err)

	client := testutil.ConnectInMemory(s.T(), handler)

	result, err := client.Complete(context.Background(), &mcp.CompleteParams{
		Ref: &mcp.CompleteReference{
			Type: "ref/prompt",
			Name: "greet",
		},
		Argument: mcp.CompleteParamsArgument{
			Name: "language",
		},
	})

	s.Require().NoError(err)
	s.Len(result.Completion.Values, 3)
	s.Contains(result.Completion.Values, "English")
	s.Contains(result.Completion.Values, "Spanish")
	s.Contains(result.Completion.Values, "French")
}

func (s *CompletionIntegrationSuite) TestCompletionError() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("completion-test"),
		mcpio.WithCompletion(func(ctx context.Context, reqCtx mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
			// Return error for all completion requests
			return nil, mcpio.NewCompletionError("completion not available")
		}),
	)
	s.Require().NoError(err)

	client := testutil.ConnectInMemory(s.T(), handler)

	_, err = client.Complete(context.Background(), &mcp.CompleteParams{
		Ref: &mcp.CompleteReference{Type: "ref/prompt"},
	})

	s.Require().Error(err)
	s.Contains(err.Error(), "completion not available")
}

func (s *CompletionIntegrationSuite) TestCompletionWithURI() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("completion-test"),
		mcpio.WithCompletion(func(ctx context.Context, reqCtx mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
			// Verify URI field is populated for ref/resource
			if ref.Type == "ref/resource" && ref.URI == "file:///data/" {
				return &mcpio.CompletionResult{
					Values: []string{"file:///data/users.json", "file:///data/products.json"},
				}, nil
			}
			return &mcpio.CompletionResult{Values: []string{}}, nil
		}),
	)
	s.Require().NoError(err)

	client := testutil.ConnectInMemory(s.T(), handler)

	result, err := client.Complete(context.Background(), &mcp.CompleteParams{
		Ref: &mcp.CompleteReference{
			Type: "ref/resource",
			URI:  "file:///data/",
		},
	})

	s.Require().NoError(err)
	s.Require().Len(result.Completion.Values, 2)
	s.Contains(result.Completion.Values, "file:///data/users.json")
}

func (s *CompletionIntegrationSuite) TestCompletionWithValue() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("completion-test"),
		mcpio.WithCompletion(func(ctx context.Context, reqCtx mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
			// Filter based on current input value
			if ref.Value == "Eng" {
				return &mcpio.CompletionResult{
					Values: []string{"English"},
				}, nil
			}
			return &mcpio.CompletionResult{
				Values: []string{"English", "Spanish", "French"},
			}, nil
		}),
	)
	s.Require().NoError(err)

	client := testutil.ConnectInMemory(s.T(), handler)

	result, err := client.Complete(context.Background(), &mcp.CompleteParams{
		Ref: &mcp.CompleteReference{Type: "ref/prompt"},
		Argument: mcp.CompleteParamsArgument{
			Name:  "language",
			Value: "Eng",
		},
	})

	s.Require().NoError(err)
	s.Require().Len(result.Completion.Values, 1)
	s.Equal("English", result.Completion.Values[0])
}

func (s *CompletionIntegrationSuite) TestCompletionWithContext() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("completion-test"),
		mcpio.WithCompletion(func(ctx context.Context, reqCtx mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
			// Provide context-aware suggestions
			if ref.Argument == "style" && ref.Context["language"] == "Spanish" {
				return &mcpio.CompletionResult{
					Values: []string{"formal", "informal"},
				}, nil
			}
			return &mcpio.CompletionResult{
				Values: []string{"casual", "professional"},
			}, nil
		}),
	)
	s.Require().NoError(err)

	client := testutil.ConnectInMemory(s.T(), handler)

	result, err := client.Complete(context.Background(), &mcp.CompleteParams{
		Ref: &mcp.CompleteReference{Type: "ref/prompt"},
		Argument: mcp.CompleteParamsArgument{
			Name: "style",
		},
		Context: &mcp.CompleteContext{
			Arguments: map[string]string{
				"language": "Spanish",
			},
		},
	})

	s.Require().NoError(err)
	s.Require().Len(result.Completion.Values, 2)
	s.Contains(result.Completion.Values, "formal")
	s.Contains(result.Completion.Values, "informal")
}

func (s *CompletionIntegrationSuite) TestCompletionWithMeta() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("completion-test"),
		mcpio.WithCompletion(func(ctx context.Context, reqCtx mcpio.RequestContext, ref mcpio.CompletionRef) (*mcpio.CompletionResult, error) {
			return &mcpio.CompletionResult{
				Values: []string{"option1", "option2"},
				Meta: map[string]any{
					"source":    "cache",
					"timestamp": "2025-01-01T00:00:00Z",
				},
			}, nil
		}),
	)
	s.Require().NoError(err)

	client := testutil.ConnectInMemory(s.T(), handler)

	result, err := client.Complete(context.Background(), &mcp.CompleteParams{
		Ref: &mcp.CompleteReference{Type: "ref/prompt"},
	})

	s.Require().NoError(err)
	s.NotNil(result.Meta)
	s.Equal("cache", result.Meta["source"])
	s.Equal("2025-01-01T00:00:00Z", result.Meta["timestamp"])
}
