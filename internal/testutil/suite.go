package testutil

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/stretchr/testify/suite"
)

// IntegrationSuite provides common setup and teardown for integration tests
type IntegrationSuite struct {
	suite.Suite
	ProjectRoot string
	Ctx         context.Context
}

// SetupSuite builds the example binaries needed for integration testing
func (s *IntegrationSuite) SetupSuite() {
	// Get project root
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(basepath, "..", "..")

	// Build the example binaries for integration testing
	buildCmd := exec.Command("make", "build-examples")
	buildCmd.Dir = s.ProjectRoot
	buildOutput, err := buildCmd.CombinedOutput()
	s.Require().NoError(err, "Failed to build example binaries: %s", string(buildOutput))
}

// TearDownSuite cleans up built binaries
func (s *IntegrationSuite) TearDownSuite() {
	cleanCmd := exec.Command("make", "clean")
	cleanCmd.Dir = s.ProjectRoot
	if err := cleanCmd.Run(); err != nil {
		// Log but don't fail on cleanup errors
		if s.T() != nil {
			s.T().Logf("cleanup error (ignored): %v", err)
		}
	}
}

// SetupTest initializes the context for each test
func (s *IntegrationSuite) SetupTest() {
	s.Ctx = context.Background()
}
