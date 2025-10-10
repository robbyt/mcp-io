package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	mcpio "github.com/robbyt/mcp-io"
)

// BasicConfig represents the initial system configuration
type BasicConfig struct {
	SystemName       string `json:"systemName"       jsonschema:"description:Name of the system"`
	Environment      string `json:"environment"      jsonschema:"description:Target environment,enum:development,enum:staging,enum:production"`
	EnableMonitoring bool   `json:"enableMonitoring" jsonschema:"description:Enable system monitoring"`
}

// AdvancedConfig represents additional configuration for staging/production
type AdvancedConfig struct {
	DatabaseURL    string `json:"databaseUrl"    jsonschema:"description:Database connection URL"`
	CacheEnabled   bool   `json:"cacheEnabled"   jsonschema:"description:Enable Redis caching"`
	APIRateLimit   int    `json:"apiRateLimit"   jsonschema:"description:API requests per minute,minimum:100,maximum:10000"`
	BackupSchedule string `json:"backupSchedule" jsonschema:"description:Backup schedule in cron format"`
}

// DeploymentConfig represents deployment-specific settings for production
type DeploymentConfig struct {
	Region       string `json:"region"       jsonschema:"description:AWS region,enum:us-east-1,enum:us-west-2,enum:eu-west-1"`
	InstanceType string `json:"instanceType" jsonschema:"description:EC2 instance type,enum:t3.micro,enum:t3.small,enum:t3.medium,enum:t3.large"`
	AutoScaling  bool   `json:"autoScaling"  jsonschema:"description:Enable auto-scaling"`
	MinInstances int    `json:"minInstances" jsonschema:"description:Minimum number of instances,minimum:1,maximum:10"`
	MaxInstances int    `json:"maxInstances" jsonschema:"description:Maximum number of instances,minimum:1,maximum:50"`
}

// configureSystem demonstrates a comprehensive multi-step elicitation workflow
func configureSystem(ctx context.Context, _ struct{}) (map[string]any, error) {
	// Step 1: Elicit basic system configuration
	basicResult, err := mcpio.ElicitTyped[BasicConfig](ctx,
		"Step 1/3: Configure basic system settings")
	if err != nil {
		return nil, fmt.Errorf("failed to elicit basic configuration: %w", err)
	}

	if !basicResult.IsAccepted() {
		return map[string]any{
			"status": "cancelled",
			"step":   "basic_config",
			"reason": fmt.Sprintf("User %s the basic configuration", basicResult.Action),
		}, nil
	}

	// Parse basic configuration
	var basicConfig BasicConfig
	if err := basicResult.DecodeContent(&basicConfig); err != nil {
		return nil, fmt.Errorf("failed to parse basic config: %w", err)
	}

	response := map[string]any{
		"status":      "configured",
		"basicConfig": basicConfig,
	}

	// Step 2: Conditionally elicit advanced configuration for staging/production
	var advancedConfig *AdvancedConfig
	if basicConfig.Environment == "staging" || basicConfig.Environment == "production" {
		advancedResult, err := mcpio.ElicitTyped[AdvancedConfig](ctx,
			fmt.Sprintf("Step 2/3: Configure advanced settings for %s environment", basicConfig.Environment))
		if err != nil {
			return nil, fmt.Errorf("failed to elicit advanced configuration: %w", err)
		}

		if advancedResult.IsAccepted() {
			advancedConfig = &AdvancedConfig{}
			if err := advancedResult.DecodeContent(advancedConfig); err != nil {
				return nil, fmt.Errorf("failed to parse advanced config: %w", err)
			}
			response["advancedConfig"] = advancedConfig
		} else {
			return map[string]any{
				"status":      "partial",
				"step":        "advanced_config",
				"reason":      fmt.Sprintf("User %s the advanced configuration", advancedResult.Action),
				"basicConfig": basicConfig,
			}, nil
		}
	}

	// Step 3: Conditionally elicit deployment configuration for production
	var deploymentConfig *DeploymentConfig
	if basicConfig.Environment == "production" {
		deployResult, err := mcpio.ElicitTyped[DeploymentConfig](ctx,
			"Step 3/3: Configure production deployment settings")
		if err != nil {
			return nil, fmt.Errorf("failed to elicit deployment configuration: %w", err)
		}

		if deployResult.IsAccepted() {
			deploymentConfig = &DeploymentConfig{}
			if err := deployResult.DecodeContent(deploymentConfig); err != nil {
				return nil, fmt.Errorf("failed to parse deployment config: %w", err)
			}

			// Validate deployment configuration
			if deploymentConfig.MinInstances > deploymentConfig.MaxInstances {
				return map[string]any{
					"status": "error",
					"reason": "Minimum instances cannot be greater than maximum instances",
				}, nil
			}

			response["deploymentConfig"] = deploymentConfig
		} else {
			return map[string]any{
				"status":         "partial",
				"step":           "deployment_config",
				"reason":         fmt.Sprintf("User %s the deployment configuration", deployResult.Action),
				"basicConfig":    basicConfig,
				"advancedConfig": advancedConfig,
			}, nil
		}
	}

	// Step 4: Final confirmation with summary
	summary := generateConfigSummary(basicConfig, advancedConfig, deploymentConfig)
	confirmResult, err := mcpio.ElicitSimple(ctx,
		fmt.Sprintf("Configuration Summary:\n%s\n\nProceed with this configuration?", summary),
		"confirm", "Type 'yes' to confirm or 'no' to cancel")
	if err != nil {
		return nil, fmt.Errorf("failed to get confirmation: %w", err)
	}

	if !confirmResult.IsAccepted() {
		return map[string]any{
			"status": "cancelled",
			"step":   "confirmation",
			"reason": "User did not confirm the configuration",
		}, nil
	}

	confirmation := ""
	if content := confirmResult.GetContent(); content != nil {
		if val, ok := content["confirm"].(string); ok {
			confirmation = val
		}
	}

	if confirmation != "yes" {
		return map[string]any{
			"status": "cancelled",
			"step":   "confirmation",
			"reason": "User did not confirm with 'yes'",
		}, nil
	}

	// Mark as fully configured
	response["status"] = "fully_configured"
	response["steps_completed"] = getCompletedSteps(basicConfig, advancedConfig, deploymentConfig)
	response["message"] = fmt.Sprintf("System '%s' successfully configured for %s environment",
		basicConfig.SystemName, basicConfig.Environment)

	return response, nil
}

// generateConfigSummary creates a human-readable summary of the configuration
func generateConfigSummary(basic BasicConfig, advanced *AdvancedConfig, deployment *DeploymentConfig) string {
	summary := fmt.Sprintf(`Basic Configuration:
  • System Name: %s
  • Environment: %s
  • Monitoring: %t`, basic.SystemName, basic.Environment, basic.EnableMonitoring)

	if advanced != nil {
		summary += fmt.Sprintf(`

Advanced Configuration:
  • Database URL: %s
  • Cache Enabled: %t
  • API Rate Limit: %d req/min
  • Backup Schedule: %s`, advanced.DatabaseURL, advanced.CacheEnabled, advanced.APIRateLimit, advanced.BackupSchedule)
	}

	if deployment != nil {
		summary += fmt.Sprintf(`

Deployment Configuration:
  • Region: %s
  • Instance Type: %s
  • Auto-scaling: %t
  • Instances: %d-%d`, deployment.Region, deployment.InstanceType, deployment.AutoScaling,
			deployment.MinInstances, deployment.MaxInstances)
	}

	return summary
}

// getCompletedSteps returns the number of configuration steps completed
func getCompletedSteps(basic BasicConfig, advanced *AdvancedConfig, deployment *DeploymentConfig) int {
	steps := 1 // Basic config always completed if we get here
	if advanced != nil {
		steps++
	}
	if deployment != nil {
		steps++
	}
	return steps
}

func main() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("multistep-config-server"),
		mcpio.WithVersion("1.0.0"),

		// Multi-step elicitation tool
		mcpio.WithTool("configure_system", "Multi-step system configuration with conditional elicitation", configureSystem),
	)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	// Start HTTP server
	http.Handle("/mcp", handler)
	log.Printf("Multi-step elicitation server listening on :8080/mcp")
	log.Printf("Test with: mcp call configure_system --params '{}' http://localhost:8080/mcp")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
