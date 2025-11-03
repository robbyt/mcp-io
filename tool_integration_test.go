//go:build integration

package mcpio_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type ToolIntegrationTestSuite struct {
	testutil.IntegrationSuite
}

func TestToolIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ToolIntegrationTestSuite))
}

func (s *ToolIntegrationTestSuite) TestProgressNotificationIntegration() {
	s.Run("BasicProgress", func() {
		// Capture progress notifications
		var notifications []*mcp.ProgressNotificationParams

		// Create handler
		handler, err := mcpio.NewHandler(
			mcpio.WithName("progress-test"),
			mcpio.WithTool("process", "Test basic progress", func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
				Items int `json:"items"`
			}) (struct {
				Result string `json:"result"`
			}, error,
			) {
				session := toolCtx.GetSession()
				total := float64(input.Items)

				// Send progress notifications
				for i := 0; i < input.Items; i++ {
					if err := session.NotifyProgress(ctx, float64(i+1), total); err != nil {
						return struct {
							Result string `json:"result"`
						}{}, err
					}
				}

				return struct {
					Result string `json:"result"`
				}{Result: "done"}, nil
			}),
		)
		s.Require().NoError(err)

		// Set up client with progress handler
		clientTransport, serverTransport := mcp.NewInMemoryTransports()
		server := handler.GetServer()
		mcpServer, ok := server.Unwrap().(*mcp.Server)
		s.Require().True(ok)

		go func() {
			_ = mcpServer.Run(s.Ctx, serverTransport) //nolint:errcheck
		}()

		client := mcp.NewClient(testutil.DefaultTestImplementation(), &mcp.ClientOptions{
			ProgressNotificationHandler: func(ctx context.Context, req *mcp.ProgressNotificationClientRequest) {
				notifications = append(notifications, req.Params)
			},
		})
		session, err := client.Connect(s.Ctx, clientTransport, nil)
		s.Require().NoError(err)
		defer session.Close() //nolint:errcheck

		// Call tool
		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name:      "process",
			Arguments: map[string]any{"items": 3},
		})
		s.Require().NoError(err)
		s.False(result.IsError)

		// Verify notifications were received
		s.Require().Len(notifications, 3)
		s.InEpsilon(1.0, notifications[0].Progress, 0.0001)
		s.InEpsilon(2.0, notifications[1].Progress, 0.0001)
		s.InEpsilon(3.0, notifications[2].Progress, 0.0001)
		s.InEpsilon(3.0, notifications[0].Total, 0.0001)
	})

	s.Run("WithToken", func() {
		var notifications []*mcp.ProgressNotificationParams

		handler, err := mcpio.NewHandler(
			mcpio.WithName("progress-test"),
			mcpio.WithTool("process", "Test progress with token", func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
				Count int `json:"count"`
			}) (struct {
				Result string `json:"result"`
			}, error,
			) {
				session := toolCtx.GetSession()
				token := toolCtx.GetProgressToken()

				// Send progress with token
				for i := 0; i < input.Count; i++ {
					if err := session.NotifyProgress(ctx, float64(i+1), float64(input.Count),
						capabilities.WithProgressToken(token)); err != nil {
						return struct {
							Result string `json:"result"`
						}{}, err
					}
				}

				return struct {
					Result string `json:"result"`
				}{Result: "done"}, nil
			}),
		)
		s.Require().NoError(err)

		clientTransport, serverTransport := mcp.NewInMemoryTransports()
		server := handler.GetServer()
		mcpServer, ok := server.Unwrap().(*mcp.Server)
		s.Require().True(ok)

		go func() {
			_ = mcpServer.Run(s.Ctx, serverTransport) //nolint:errcheck
		}()

		client := mcp.NewClient(testutil.DefaultTestImplementation(), &mcp.ClientOptions{
			ProgressNotificationHandler: func(ctx context.Context, req *mcp.ProgressNotificationClientRequest) {
				notifications = append(notifications, req.Params)
			},
		})
		session, err := client.Connect(s.Ctx, clientTransport, nil)
		s.Require().NoError(err)
		defer session.Close() //nolint:errcheck

		// Call tool with progress token in meta
		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name:      "process",
			Arguments: map[string]any{"count": 2},
			Meta:      mcp.Meta{"progressToken": "test-token-123"},
		})
		s.Require().NoError(err)
		s.False(result.IsError)

		// Verify token was echoed back
		s.Require().Len(notifications, 2)
		s.Equal("test-token-123", notifications[0].ProgressToken)
		s.Equal("test-token-123", notifications[1].ProgressToken)
	})

	s.Run("WithMessage", func() {
		var notifications []*mcp.ProgressNotificationParams

		handler, err := mcpio.NewHandler(
			mcpio.WithName("progress-test"),
			mcpio.WithTool("process", "Test progress with message", func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
				Files []string `json:"files"`
			}) (struct {
				Result string `json:"result"`
			}, error,
			) {
				session := toolCtx.GetSession()
				total := float64(len(input.Files))

				for i, file := range input.Files {
					msg := fmt.Sprintf("Processing %s (%d/%d)", file, i+1, len(input.Files))
					if err := session.NotifyProgress(ctx, float64(i+1), total,
						capabilities.WithProgressMessage(msg)); err != nil {
						return struct {
							Result string `json:"result"`
						}{}, err
					}
				}

				return struct {
					Result string `json:"result"`
				}{Result: "done"}, nil
			}),
		)
		s.Require().NoError(err)

		clientTransport, serverTransport := mcp.NewInMemoryTransports()
		server := handler.GetServer()
		mcpServer, ok := server.Unwrap().(*mcp.Server)
		s.Require().True(ok)

		go func() {
			_ = mcpServer.Run(s.Ctx, serverTransport) //nolint:errcheck
		}()

		client := mcp.NewClient(testutil.DefaultTestImplementation(), &mcp.ClientOptions{
			ProgressNotificationHandler: func(ctx context.Context, req *mcp.ProgressNotificationClientRequest) {
				notifications = append(notifications, req.Params)
			},
		})
		session, err := client.Connect(s.Ctx, clientTransport, nil)
		s.Require().NoError(err)
		defer session.Close() //nolint:errcheck

		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name:      "process",
			Arguments: map[string]any{"files": []string{"file1.txt", "file2.txt"}},
		})
		s.Require().NoError(err)
		s.False(result.IsError)

		// Verify messages
		s.Require().Len(notifications, 2)
		s.Equal("Processing file1.txt (1/2)", notifications[0].Message)
		s.Equal("Processing file2.txt (2/2)", notifications[1].Message)
	})

	s.Run("AllOptions", func() {
		var notifications []*mcp.ProgressNotificationParams

		handler, err := mcpio.NewHandler(
			mcpio.WithName("progress-test"),
			mcpio.WithTool("process", "Test progress with all options", func(ctx context.Context, toolCtx mcpio.RequestContext, input struct{}) (struct {
				Result string `json:"result"`
			}, error,
			) {
				session := toolCtx.GetSession()
				token := toolCtx.GetProgressToken()

				if err := session.NotifyProgress(ctx, 1.0, 2.0,
					capabilities.WithProgressToken(token),
					capabilities.WithProgressMessage("Processing step 1"),
					capabilities.WithProgressMeta(map[string]any{"step": "analyze"})); err != nil {
					return struct {
						Result string `json:"result"`
					}{}, err
				}

				if err := session.NotifyProgress(ctx, 2.0, 2.0,
					capabilities.WithProgressToken(token),
					capabilities.WithProgressMessage("Processing step 2"),
					capabilities.WithProgressMeta(map[string]any{"step": "finalize"})); err != nil {
					return struct {
						Result string `json:"result"`
					}{}, err
				}

				return struct {
					Result string `json:"result"`
				}{Result: "done"}, nil
			}),
		)
		s.Require().NoError(err)

		clientTransport, serverTransport := mcp.NewInMemoryTransports()
		server := handler.GetServer()
		mcpServer, ok := server.Unwrap().(*mcp.Server)
		s.Require().True(ok)

		go func() {
			_ = mcpServer.Run(s.Ctx, serverTransport) //nolint:errcheck
		}()

		client := mcp.NewClient(testutil.DefaultTestImplementation(), &mcp.ClientOptions{
			ProgressNotificationHandler: func(ctx context.Context, req *mcp.ProgressNotificationClientRequest) {
				notifications = append(notifications, req.Params)
			},
		})
		session, err := client.Connect(s.Ctx, clientTransport, nil)
		s.Require().NoError(err)
		defer session.Close() //nolint:errcheck

		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name:      "process",
			Arguments: map[string]any{},
			Meta:      mcp.Meta{"progressToken": 789},
		})
		s.Require().NoError(err)
		s.False(result.IsError)

		// Verify all fields
		s.Require().Len(notifications, 2)

		// First notification
		s.Equal(float64(789), notifications[0].ProgressToken) //nolint:testifylint
		s.Equal("Processing step 1", notifications[0].Message)
		s.InEpsilon(1.0, notifications[0].Progress, 0.0001)
		s.InEpsilon(2.0, notifications[0].Total, 0.0001)
		s.Equal("analyze", notifications[0].Meta["step"])

		// Second notification
		s.Equal(float64(789), notifications[1].ProgressToken) //nolint:testifylint
		s.Equal("Processing step 2", notifications[1].Message)
		s.InEpsilon(2.0, notifications[1].Progress, 0.0001)
		s.InEpsilon(2.0, notifications[1].Total, 0.0001)
		s.Equal("finalize", notifications[1].Meta["step"])
	})

	s.Run("ConcurrentRequests", func() {
		var notifications []*mcp.ProgressNotificationParams
		var notificationsMu sync.Mutex

		handler, err := mcpio.NewHandler(
			mcpio.WithName("progress-test"),
			mcpio.WithTool("process", "Test concurrent progress", func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
				ID    string `json:"id"`
				Steps int    `json:"steps"`
			}) (struct {
				Result string `json:"result"`
			}, error,
			) {
				session := toolCtx.GetSession()
				token := toolCtx.GetProgressToken()

				for i := 0; i < input.Steps; i++ {
					if err := session.NotifyProgress(ctx, float64(i+1), float64(input.Steps),
						capabilities.WithProgressToken(token),
						capabilities.WithProgressMessage(fmt.Sprintf("%s: step %d", input.ID, i+1))); err != nil {
						return struct {
							Result string `json:"result"`
						}{}, err
					}
				}

				return struct {
					Result string `json:"result"`
				}{Result: input.ID}, nil
			}),
		)
		s.Require().NoError(err)

		clientTransport, serverTransport := mcp.NewInMemoryTransports()
		server := handler.GetServer()
		mcpServer, ok := server.Unwrap().(*mcp.Server)
		s.Require().True(ok)

		go func() {
			_ = mcpServer.Run(s.Ctx, serverTransport) //nolint:errcheck
		}()

		client := mcp.NewClient(testutil.DefaultTestImplementation(), &mcp.ClientOptions{
			ProgressNotificationHandler: func(ctx context.Context, req *mcp.ProgressNotificationClientRequest) {
				notificationsMu.Lock()
				defer notificationsMu.Unlock()
				notifications = append(notifications, req.Params)
			},
		})
		session, err := client.Connect(s.Ctx, clientTransport, nil)
		s.Require().NoError(err)
		defer session.Close() //nolint:errcheck

		// Make concurrent calls with different tokens
		var wg sync.WaitGroup
		tokens := []any{"token-1", "token-2", "token-3"}

		for i, token := range tokens {
			wg.Add(1)
			go func(idx int, tok any) {
				defer wg.Done()
				_, _ = session.CallTool(s.Ctx, &mcp.CallToolParams{ //nolint:errcheck
					Name: "process",
					Arguments: map[string]any{
						"id":    fmt.Sprintf("request-%d", idx+1),
						"steps": 2,
					},
					Meta: mcp.Meta{"progressToken": tok},
				})
			}(i, token)
		}

		wg.Wait()

		// Verify we got notifications for all tokens
		notificationsMu.Lock()
		defer notificationsMu.Unlock()

		s.Require().Len(notifications, 6) // 3 requests × 2 steps each

		// Verify each token appears twice
		tokenCounts := make(map[any]int)
		for _, notif := range notifications {
			tokenCounts[notif.ProgressToken]++
		}

		s.Equal(2, tokenCounts["token-1"])
		s.Equal(2, tokenCounts["token-2"])
		s.Equal(2, tokenCounts["token-3"])
	})
}

func (s *ToolIntegrationTestSuite) TestToolHandlerIntegration() {
	s.Run("ToUpper", func() {
		// Create the server with the same tools as cli-tool
		handler, err := mcpio.NewHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("to_upper", "Convert text to uppercase", func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
				Text string `json:"text"`
			}) (struct {
				Result string `json:"result"`
			}, error,
			) {
				return struct {
					Result string `json:"result"`
				}{Result: "HELLO WORLD"}, nil
			}),
		)
		s.Require().NoError(err)

		// Connect client using testutil helper
		session := testutil.ConnectInMemory(s.T(), handler)

		// Call the to_upper tool
		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name:      "to_upper",
			Arguments: map[string]any{"text": "hello world"},
		})
		s.Require().NoError(err)

		s.False(result.IsError)
		s.Len(result.Content, 1)
	})

	s.Run("CountWords", func() {
		// Create server with count tool
		handler, err := mcpio.NewHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("count", "Count words or characters in text", func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
				Text string `json:"text"`
				Type string `json:"type"`
			}) (struct {
				Count int    `json:"count"`
				Type  string `json:"type"`
			}, error,
			) {
				count := 0
				if input.Type == "words" && input.Text == "this is a test" {
					count = 4
				}
				return struct {
					Count int    `json:"count"`
					Type  string `json:"type"`
				}{Count: count, Type: input.Type}, nil
			}),
		)
		s.Require().NoError(err)

		// Connect client using testutil helper
		session := testutil.ConnectInMemory(s.T(), handler)

		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name: "count",
			Arguments: map[string]any{
				"text": "this is a test",
				"type": "words",
			},
		})
		s.Require().NoError(err)
		s.False(result.IsError)
	})

	s.Run("ValidationError", func() {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("text-processor"),
			mcpio.WithTool("count", "Count words or characters in text", func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
				Text string `json:"text"`
				Type string `json:"type"`
			}) (struct {
				Count int    `json:"count"`
				Type  string `json:"type"`
			}, error,
			) {
				if input.Type != "words" && input.Type != "chars" {
					return struct {
						Count int    `json:"count"`
						Type  string `json:"type"`
					}{}, mcpio.NewToolError("unsupported count type: " + input.Type)
				}
				return struct {
					Count int    `json:"count"`
					Type  string `json:"type"`
				}{Count: 0, Type: input.Type}, nil
			}),
		)
		s.Require().NoError(err)

		// Connect client using testutil helper
		session := testutil.ConnectInMemory(s.T(), handler)

		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name: "count",
			Arguments: map[string]any{
				"text": "this is a test",
				"type": "invalid_type",
			},
		})
		s.Require().NoError(err)

		s.True(result.IsError)
		s.Len(result.Content, 1)
		if textContent, ok := result.Content[0].(*mcp.TextContent); s.True(ok) {
			s.Contains(textContent.Text, "unsupported count type:")
		}
	})

	s.Run("RawToolErrorCodePreservation", func() {
		// Create handler with raw tool that returns errors with codes
		handler, err := mcpio.NewHandler(
			mcpio.WithName("error-test"),
			mcpio.WithRawTool("validate", "Test error code preservation",
				`{"type":"object","properties":{"input":{"type":"string"}}}`,
				func(ctx context.Context, toolCtx mcpio.RequestContext, input []byte) ([]byte, error) {
					// Return an error with code
					return nil, mcpio.NewToolErrorWithCode("validation failed", mcpio.ErrorCodeValidation)
				},
			),
		)
		s.Require().NoError(err)

		// Connect client using testutil helper
		session := testutil.ConnectInMemory(s.T(), handler)

		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name:      "validate",
			Arguments: map[string]any{"input": "test"},
		})
		s.Require().NoError(err)

		// Verify error code is preserved in the error message
		s.True(result.IsError, "result should be marked as error")
		s.Require().Len(result.Content, 1, "result should have exactly one content item")
		s.Require().IsType(&mcp.TextContent{}, result.Content[0], "content should be TextContent")
		textContent := result.Content[0].(*mcp.TextContent)
		s.Contains(textContent.Text, "[VALIDATION_ERROR]", "error message should include error code")
		s.Contains(textContent.Text, "validation failed", "error message should include original message")
	})

	s.Run("RawToolErrorWithoutCode", func() {
		// Create handler with raw tool that returns errors without codes
		handler, err := mcpio.NewHandler(
			mcpio.WithName("error-test"),
			mcpio.WithRawTool("process", "Test error without code",
				`{"type":"object","properties":{"input":{"type":"string"}}}`,
				func(ctx context.Context, toolCtx mcpio.RequestContext, input []byte) ([]byte, error) {
					// Return an error without code
					return nil, mcpio.NewToolError("processing failed")
				},
			),
		)
		s.Require().NoError(err)

		// Connect client using testutil helper
		session := testutil.ConnectInMemory(s.T(), handler)

		result, err := session.CallTool(s.Ctx, &mcp.CallToolParams{
			Name:      "process",
			Arguments: map[string]any{"input": "test"},
		})
		s.Require().NoError(err)

		// Verify error message appears without code prefix
		s.True(result.IsError, "result should be marked as error")
		s.Require().Len(result.Content, 1, "result should have exactly one content item")
		s.Require().IsType(&mcp.TextContent{}, result.Content[0], "content should be TextContent")
		textContent := result.Content[0].(*mcp.TextContent)
		s.Equal("processing failed", textContent.Text, "error message should match exactly")
		s.NotContains(textContent.Text, "[", "error message should not have code prefix")
	})
}
