# Multi-Step Elicitation HTTP Example

A HTTP MCP server demonstrating advanced multi-step elicitation workflows. This example shows how to build conditional elicitation "question and answer" flows where subsequent steps depend on previous user responses, including validation, confirmation, and graceful error handling.

## Features Demonstrated

### 🔄 **Multi-Step Conditional Workflow**
- **Step 1**: Basic system configuration (always required)
- **Step 2**: Advanced configuration (only for staging/production environments)
- **Step 3**: Deployment settings (only for production environment)
- **Step 4**: Final confirmation with complete summary

### 🎯 **Conditional Logic**
- Environment-based elicitation: different steps for development vs production
- Validation logic: ensures deployment constraints are met
- Graceful partial completion: returns meaningful status for cancelled steps

### 📋 **Response Types**
- **`fully_configured`**: All applicable steps completed successfully
- **`partial`**: User cancelled a conditional step (advanced/deployment)
- **`cancelled`**: User cancelled required step (basic config or confirmation)
- **`error`**: Validation failed (e.g., min > max instances)

### 🔍 **Rich Schema Validation**
- Enum constraints for environment and instance types
- Numeric ranges for ports and instance counts
- Required field validation
- Format validation for URLs and schedules

## Running the Server

```bash
# Build and run
go run main.go

# Or use make if available
make build-http-multistep
./bin/http-multistep
```

The server starts on `http://localhost:8080/mcp`

## Testing the Multi-Step Workflow

**Important**: The `mcp` CLI tool does not support elicitation features. This example demonstrates multi-step elicitation workflows that require a session-aware MCP client.

### What you CAN test with mcp CLI:
```bash
# List available tools (works)
mcp tools http://localhost:8080/mcp
```

### What you CANNOT test with mcp CLI:
```bash
# This will fail - elicitation requires session support
mcp call configure_system --params '{}' http://localhost:8080/mcp
```

### To test multi-step elicitation:
- Use an MCP client that supports elicitation (like Claude Desktop)
- Write a custom test client that handles elicitation requests
- Use integration tests with mock elicitation flows

### 3. Example Elicitation Flow

#### Development Environment (Simple Flow)
```
Step 1: Basic Config
- systemName: "my-dev-app"
- environment: "development"
- enableMonitoring: false

-> Only requires basic config, skips advanced and deployment steps
```

#### Production Environment (Full Flow)
```
Step 1: Basic Config
- systemName: "my-prod-app"
- environment: "production"
- enableMonitoring: true

Step 2: Advanced Config (triggered by production)
- databaseURL: "postgresql://prod-db:5432/app"
- cacheEnabled: true
- apiRateLimit: 1000
- backupSchedule: "0 2 * * *"

Step 3: Deployment Config (triggered by production)
- region: "us-east-1"
- instanceType: "t3.medium"
- autoScaling: true
- minInstances: 2
- maxInstances: 10

Step 4: Confirmation
- Shows complete summary
- Requires "yes" to proceed
```

## Example Responses

### Successful Full Configuration
```json
{
  "status": "fully_configured",
  "basicConfig": { /* basic settings */ },
  "advancedConfig": { /* advanced settings */ },
  "deploymentConfig": { /* deployment settings */ },
  "steps_completed": 3,
  "message": "System 'my-prod-app' successfully configured for production environment"
}
```

### Partial Configuration (User Cancelled Advanced Step)
```json
{
  "status": "partial",
  "step": "advanced_config",
  "reason": "User decline the advanced configuration",
  "basicConfig": { /* basic settings */ }
}
```

### Validation Error
```json
{
  "status": "error",
  "reason": "Minimum instances cannot be greater than maximum instances"
}
```

## Key Implementation Patterns

### Conditional Elicitation
```go
// Only request advanced config for staging/production
if basicConfig.Environment == "staging" || basicConfig.Environment == "production" {
    advancedResult, err := mcpio.ElicitTyped[AdvancedConfig](ctx, capability,
        "Step 2/3: Configure advanced settings...")
    // Handle response...
}
```

### Response Validation
```go
// Validate deployment configuration
if deploymentConfig.MinInstances > deploymentConfig.MaxInstances {
    return map[string]any{
        "status": "error",
        "reason": "Minimum instances cannot be greater than maximum instances",
    }, nil
}
```

### Graceful Error Handling
```go
// Track progress and provide meaningful responses
if !result.IsAccepted() {
    return map[string]any{
        "status": "partial",
        "step": "deployment_config",
        "reason": fmt.Sprintf("User %s the deployment configuration", result.Action),
        "basicConfig": basicConfig,    // Preserve completed work
        "advancedConfig": advancedConfig,
    }, nil
}
```

### Summary Generation
```go
// Generate human-readable configuration summary
summary := generateConfigSummary(basicConfig, advancedConfig, deploymentConfig)
confirmResult, err := mcpio.ElicitSimple(ctx, capability,
    fmt.Sprintf("Configuration Summary:\n%s\n\nProceed?", summary),
    "confirm", "Type 'yes' to confirm")
```

## Schema Examples

The example uses rich JSON schema validation:

```go
type DeploymentConfig struct {
    Region       string `json:"region" jsonschema:"description=AWS region,enum=us-east-1,enum=us-west-2,enum=eu-west-1"`
    InstanceType string `json:"instanceType" jsonschema:"description=EC2 instance type,enum=t3.micro,enum=t3.small,enum=t3.medium,enum=t3.large"`
    MinInstances int    `json:"minInstances" jsonschema:"description=Minimum number of instances,minimum=1,maximum=10"`
    MaxInstances int    `json:"maxInstances" jsonschema:"description=Maximum number of instances,minimum=1,maximum=50"`
}
```

This generates client forms with:
- Dropdown menus for enum values
- Numeric input validation with min/max constraints
- Helpful descriptions for each field

## Educational Value

This example demonstrates production-ready patterns for:
- **Complex conditional workflows** that adapt based on user input
- **Validation and error handling** with meaningful user feedback
- **State preservation** across multi-step processes
- **User experience** with progress indicators and summaries
- **Schema design** for rich client form generation

Perfect for learning how to build sophisticated interactive MCP tools that guide users through complex configuration processes.