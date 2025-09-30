package main

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

// MockElicitationCapability for multi-step testing
type MockElicitationCapability struct {
	Responses []*mcp.ElicitResult
	CallIndex int
}

func (m *MockElicitationCapability) Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error) {
	if m.CallIndex >= len(m.Responses) {
		return &mcp.ElicitResult{Action: "cancel"}, nil
	}
	result := m.Responses[m.CallIndex]
	m.CallIndex++
	return result, nil
}

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

func (s *HttpMultistepTestSuite) TestDevelopmentFlow() {
	ctx := s.T().Context()

	// Test development environment (basic config only)
	s.Run("DevelopmentEnvironment", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"systemName":       "TestApp",
						"environment":      "development",
						"enableMonitoring": false,
					},
				},
				{
					Action: "accept",
					Content: map[string]any{
						"confirm": "yes",
					},
				},
			},
		}

		result, err := configureSystem(ctx, mockCapability, struct{}{})
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
	})
}

func (s *HttpMultistepTestSuite) TestStagingFlow() {
	ctx := s.T().Context()

	// Test staging environment (basic + advanced config)
	s.Run("StagingEnvironment", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"systemName":       "StagingApp",
						"environment":      "staging",
						"enableMonitoring": true,
					},
				},
				{
					Action: "accept",
					Content: map[string]any{
						"databaseURL":    "postgres://localhost:5432/staging",
						"cacheEnabled":   true,
						"apiRateLimit":   1000,
						"backupSchedule": "0 2 * * *",
					},
				},
				{
					Action: "accept",
					Content: map[string]any{
						"confirm": "yes",
					},
				},
			},
		}

		result, err := configureSystem(ctx, mockCapability, struct{}{})
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
	})
}

func (s *HttpMultistepTestSuite) TestProductionFlow() {
	ctx := s.T().Context()

	// Test production environment (all three configs)
	s.Run("ProductionEnvironment", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"systemName":       "ProdApp",
						"environment":      "production",
						"enableMonitoring": true,
					},
				},
				{
					Action: "accept",
					Content: map[string]any{
						"databaseURL":    "postgres://prod.db.example.com:5432/production",
						"cacheEnabled":   true,
						"apiRateLimit":   5000,
						"backupSchedule": "0 1 * * *",
					},
				},
				{
					Action: "accept",
					Content: map[string]any{
						"region":       "us-west-2",
						"instanceType": "t3.medium",
						"autoScaling":  true,
						"minInstances": 2,
						"maxInstances": 10,
					},
				},
				{
					Action: "accept",
					Content: map[string]any{
						"confirm": "yes",
					},
				},
			},
		}

		result, err := configureSystem(ctx, mockCapability, struct{}{})
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
	})
}

func (s *HttpMultistepTestSuite) TestErrorHandling() {
	ctx := s.T().Context()

	s.Run("UserCancelsBasicConfig", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{Action: "decline"},
			},
		}

		result, err := configureSystem(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Equal("basic_config", result["step"])
		s.Contains(result["reason"], "decline")
	})

	s.Run("UserCancelsAdvancedConfig", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"systemName":       "TestApp",
						"environment":      "staging",
						"enableMonitoring": true,
					},
				},
				{Action: "decline"},
			},
		}

		result, err := configureSystem(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)

		s.Equal("partial", result["status"])
		s.Equal("advanced_config", result["step"])
		s.NotNil(result["basicConfig"])
		s.Nil(result["advancedConfig"])
	})

	s.Run("InvalidDeploymentConfig", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"systemName":       "TestApp",
						"environment":      "production",
						"enableMonitoring": true,
					},
				},
				{
					Action: "accept",
					Content: map[string]any{
						"databaseURL":    "postgres://example.com/db",
						"cacheEnabled":   false,
						"apiRateLimit":   2000,
						"backupSchedule": "0 3 * * *",
					},
				},
				{
					Action: "accept",
					Content: map[string]any{
						"region":       "us-east-1",
						"instanceType": "t3.small",
						"autoScaling":  false,
						"minInstances": 5, // Invalid: min > max
						"maxInstances": 3,
					},
				},
			},
		}

		result, err := configureSystem(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)

		s.Equal("error", result["status"])
		s.Contains(result["reason"], "Minimum instances cannot be greater than maximum instances")
	})

	s.Run("UserDoesNotConfirm", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"systemName":       "TestApp",
						"environment":      "development",
						"enableMonitoring": false,
					},
				},
				{
					Action: "accept",
					Content: map[string]any{
						"confirm": "no",
					},
				},
			},
		}

		result, err := configureSystem(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Equal("confirmation", result["step"])
		s.Contains(result["reason"], "did not confirm with 'yes'")
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
	// Test that we can create the server
	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("multistep-config-server"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithSessionTool("configure_system", "Multi-step system configuration with conditional elicitation", configureSystem),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	// Verify server creation works
	server, err := serverBuilder()
	s.Require().NoError(err)
	s.NotNil(server)
}

func (s *HttpMultistepTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
