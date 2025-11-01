package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// HttpMultistepTestSuite tests the http_multistep example
type HttpMultistepTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *HttpMultistepTestSuite) SetupSuite() {
	// Get project root - we're in examples/http_multistep
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "http_multistep"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()
}

func TestHttpMultistepSuite(t *testing.T) {
	suite.Run(t, new(HttpMultistepTestSuite))
}

// mockToolContext creates a minimal ToolContext for testing with the given session
func mockToolContext(session *capabilities.Session) mcpio.RequestContext {
	return testutil.NewMockToolContext(session)
}

func (s *HttpMultistepTestSuite) TestDevelopmentFlow() {
	// Test development environment (basic config only)
	s.Run("DevelopmentEnvironment", func() {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()

		// Expect two elicit calls: basic config and confirmation
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"systemName":       "TestApp",
				"environment":      "development",
				"enableMonitoring": false,
			},
		}, nil).Once()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"confirm": "yes",
			},
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := configureSystem(ctx, mockToolContext(mockSession.Session), struct{}{})
		s.Require().NoError(err)

		s.Equal("fully_configured", result["status"])
		s.Equal(1, result["steps_completed"])
		s.Contains(result["message"], "TestApp")
		s.Contains(result["message"], "development")

		basicConfig := result["basicConfig"].(BasicConfig)
		s.Equal("TestApp", basicConfig.SystemName)
		s.Equal("development", basicConfig.Environment)
		s.False(basicConfig.EnableMonitoring)

		// Should not have advanced or deployment config for development
		s.Nil(result["advancedConfig"])
		s.Nil(result["deploymentConfig"])

		mockSession.AssertExpectations(s.T())
	})
}

func (s *HttpMultistepTestSuite) TestStagingFlow() {
	// Test staging environment (basic + advanced config)
	s.Run("StagingEnvironment", func() {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()

		// Expect three elicit calls: basic, advanced, and confirmation
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"systemName":       "StagingApp",
				"environment":      "staging",
				"enableMonitoring": true,
			},
		}, nil).Once()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"databaseURL":    "postgres://localhost:5432/staging",
				"cacheEnabled":   true,
				"apiRateLimit":   1000,
				"backupSchedule": "0 2 * * *",
			},
		}, nil).Once()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"confirm": "yes",
			},
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := configureSystem(ctx, mockToolContext(mockSession.Session), struct{}{})
		s.Require().NoError(err)

		s.Equal("fully_configured", result["status"])
		s.Equal(2, result["steps_completed"])

		basicConfig := result["basicConfig"].(BasicConfig)
		s.Equal("StagingApp", basicConfig.SystemName)
		s.Equal("staging", basicConfig.Environment)
		s.True(basicConfig.EnableMonitoring)

		// Should have advanced config for staging
		s.NotNil(result["advancedConfig"])
		advancedConfig := *result["advancedConfig"].(*AdvancedConfig)
		s.Equal("postgres://localhost:5432/staging", advancedConfig.DatabaseURL)
		s.True(advancedConfig.CacheEnabled)
		s.Equal(1000, advancedConfig.APIRateLimit)
		s.Equal("0 2 * * *", advancedConfig.BackupSchedule)

		// Should not have deployment config for staging
		s.Nil(result["deploymentConfig"])

		mockSession.AssertExpectations(s.T())
	})
}

func (s *HttpMultistepTestSuite) TestProductionFlow() {
	// Test production environment (all three configs)
	s.Run("ProductionEnvironment", func() {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()

		// Expect four elicit calls: basic, advanced, deployment, and confirmation
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"systemName":       "ProdApp",
				"environment":      "production",
				"enableMonitoring": true,
			},
		}, nil).Once()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"databaseURL":    "postgres://prod.db.example.com:5432/production",
				"cacheEnabled":   true,
				"apiRateLimit":   5000,
				"backupSchedule": "0 1 * * *",
			},
		}, nil).Once()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"region":       "us-west-2",
				"instanceType": "t3.medium",
				"autoScaling":  true,
				"minInstances": 2,
				"maxInstances": 10,
			},
		}, nil).Once()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"confirm": "yes",
			},
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := configureSystem(ctx, mockToolContext(mockSession.Session), struct{}{})
		s.Require().NoError(err)

		s.Equal("fully_configured", result["status"])
		s.Equal(3, result["steps_completed"])

		// Verify all three configs are present
		s.NotNil(result["basicConfig"])
		s.NotNil(result["advancedConfig"])
		s.NotNil(result["deploymentConfig"])

		deploymentConfig := *result["deploymentConfig"].(*DeploymentConfig)
		s.Equal("us-west-2", deploymentConfig.Region)
		s.Equal("t3.medium", deploymentConfig.InstanceType)
		s.True(deploymentConfig.AutoScaling)
		s.Equal(2, deploymentConfig.MinInstances)
		s.Equal(10, deploymentConfig.MaxInstances)

		mockSession.AssertExpectations(s.T())
	})
}

func (s *HttpMultistepTestSuite) TestErrorHandling() {
	s.Run("UserCancelsBasicConfig", func() {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "decline",
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := configureSystem(ctx, mockToolContext(mockSession.Session), struct{}{})
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Equal("basic_config", result["step"])
		s.Contains(result["reason"], "decline")

		mockSession.AssertExpectations(s.T())
	})

	s.Run("UserCancelsAdvancedConfig", func() {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"systemName":       "TestApp",
				"environment":      "staging",
				"enableMonitoring": true,
			},
		}, nil).Once()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "decline",
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := configureSystem(ctx, mockToolContext(mockSession.Session), struct{}{})
		s.Require().NoError(err)

		s.Equal("partial", result["status"])
		s.Equal("advanced_config", result["step"])
		s.NotNil(result["basicConfig"])
		s.Nil(result["advancedConfig"])

		mockSession.AssertExpectations(s.T())
	})

	s.Run("InvalidDeploymentConfig", func() {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"systemName":       "TestApp",
				"environment":      "production",
				"enableMonitoring": true,
			},
		}, nil).Once()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"databaseURL":    "postgres://example.com/db",
				"cacheEnabled":   false,
				"apiRateLimit":   2000,
				"backupSchedule": "0 3 * * *",
			},
		}, nil).Once()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"region":       "us-east-1",
				"instanceType": "t3.small",
				"autoScaling":  false,
				"minInstances": 5, // Invalid: min > max
				"maxInstances": 3,
			},
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := configureSystem(ctx, mockToolContext(mockSession.Session), struct{}{})
		s.Require().NoError(err)

		s.Equal("error", result["status"])
		s.Contains(result["reason"], "Minimum instances cannot be greater than maximum instances")

		mockSession.AssertExpectations(s.T())
	})

	s.Run("UserDoesNotConfirm", func() {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"systemName":       "TestApp",
				"environment":      "development",
				"enableMonitoring": false,
			},
		}, nil).Once()

		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"confirm": "no",
			},
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := configureSystem(ctx, mockToolContext(mockSession.Session), struct{}{})
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Equal("confirmation", result["step"])
		s.Contains(result["reason"], "did not confirm with 'yes'")

		mockSession.AssertExpectations(s.T())
	})
}

func (s *HttpMultistepTestSuite) TestHelperFunctions() {
	// Test configuration structs
	s.Run("BasicConfigStruct", func() {
		config := BasicConfig{
			SystemName:       "TestSystem",
			Environment:      "production",
			EnableMonitoring: true,
		}
		s.Equal("TestSystem", config.SystemName)
		s.Equal("production", config.Environment)
		s.True(config.EnableMonitoring)
	})

	// Test summary generation
	s.Run("ConfigSummaryGeneration", func() {
		basic := BasicConfig{
			SystemName:       "TestSystem",
			Environment:      "production",
			EnableMonitoring: true,
		}
		advanced := &AdvancedConfig{
			DatabaseURL:    "postgres://example.com/db",
			CacheEnabled:   true,
			APIRateLimit:   3000,
			BackupSchedule: "0 2 * * *",
		}
		deployment := &DeploymentConfig{
			Region:       "us-west-2",
			InstanceType: "t3.large",
			AutoScaling:  true,
			MinInstances: 2,
			MaxInstances: 8,
		}

		summary := generateConfigSummary(basic, advanced, deployment)
		s.Contains(summary, "TestSystem")
		s.Contains(summary, "production")
		s.Contains(summary, "postgres://example.com/db")
		s.Contains(summary, "us-west-2")
		s.Contains(summary, "t3.large")
	})

	// Test steps counting
	s.Run("StepsCompleted", func() {
		basic := BasicConfig{}

		// Only basic config
		steps := getCompletedSteps(basic, nil, nil)
		s.Equal(1, steps)

		// Basic + advanced
		advanced := &AdvancedConfig{}
		steps = getCompletedSteps(basic, advanced, nil)
		s.Equal(2, steps)

		// All three
		deployment := &DeploymentConfig{}
		steps = getCompletedSteps(basic, advanced, deployment)
		s.Equal(3, steps)
	})
}

func (s *HttpMultistepTestSuite) TestServerCreation() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("multistep-config-server"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("configure_system", "Multi-step system configuration with conditional elicitation", configureSystem),
	)
	s.Require().NoError(err)
	s.NotNil(handler)
	s.NotNil(handler.GetServer())
}

func (s *HttpMultistepTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
