package main

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

// DatabaseTestSuite tests the database elicitation example
type DatabaseTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *DatabaseTestSuite) SetupSuite() {
	// Get project root - we're in examples/cli_elicitation
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "cli_elicitation"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()
}

func (s *DatabaseTestSuite) SetupTest() {
	// Clear database before each test
	dbMutex.Lock()
	database = make(map[string]*Record)
	dbMutex.Unlock()
}

func TestDatabaseSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}

// mockToolContext creates a mock tool context for unit tests
func mockToolContext() mcpio.RequestContext {
	return testutil.NewMockRequestContext(nil)
}

// Test standard database operations (no elicitation)

func (s *DatabaseTestSuite) TestReadRecord() {
	ctx := s.T().Context()

	// Add a test record
	testRecord := &Record{
		ID:      "test1",
		Name:    "Test User",
		Email:   "test@example.com",
		Status:  "active",
		Age:     25,
		Created: time.Now(),
		Updated: time.Now(),
	}
	database["test1"] = testRecord

	s.Run("RecordExists", func() {
		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to retrieve"`
		}{ID: "test1"}

		result, err := readRecord(ctx, mockToolContext(), input)
		s.Require().NoError(err)

		s.Equal("found", result["status"])
		record := result["record"].(*Record)
		s.Equal("test1", record.ID)
		s.Equal("Test User", record.Name)
	})

	s.Run("RecordNotFound", func() {
		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to retrieve"`
		}{ID: "nonexistent"}

		result, err := readRecord(ctx, mockToolContext(), input)
		s.Require().NoError(err)

		s.Equal("not_found", result["status"])
		s.Equal("nonexistent", result["id"])
	})
}

func (s *DatabaseTestSuite) TestListRecords() {
	ctx := s.T().Context()

	s.Run("EmptyDatabase", func() {
		input := struct {
			Status string `json:"status,omitempty" jsonschema:"description:Optional status filter,enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
		}{}

		result, err := listRecords(ctx, mockToolContext(), input)
		s.Require().NoError(err)

		s.Equal("success", result["status"])
		s.Equal(0, result["count"])
	})

	s.Run("WithRecords", func() {
		// Add test records
		database["active1"] = &Record{ID: "active1", Status: "active", Created: time.Now()}
		database["inactive1"] = &Record{ID: "inactive1", Status: "inactive", Created: time.Now().Add(time.Second)}

		input := struct {
			Status string `json:"status,omitempty" jsonschema:"description:Optional status filter,enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
		}{}

		result, err := listRecords(ctx, mockToolContext(), input)
		s.Require().NoError(err)

		s.Equal("success", result["status"])
		s.Equal(2, result["count"])
	})

	s.Run("FilterByStatus", func() {
		// Add test records
		database["active1"] = &Record{ID: "active1", Status: "active", Created: time.Now()}
		database["active2"] = &Record{ID: "active2", Status: "active", Created: time.Now().Add(time.Second)}
		database["inactive1"] = &Record{ID: "inactive1", Status: "inactive", Created: time.Now().Add(2 * time.Second)}

		input := struct {
			Status string `json:"status,omitempty" jsonschema:"description:Optional status filter,enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
		}{Status: "active"}

		result, err := listRecords(ctx, mockToolContext(), input)
		s.Require().NoError(err)

		s.Equal("success", result["status"])
		s.Equal(2, result["count"])
		s.Equal("active", result["filter"])

		returnedRecords := result["records"].([]*Record)
		s.Len(returnedRecords, 2)
		s.Equal("active1", returnedRecords[0].ID)
	})
}

func (s *DatabaseTestSuite) TestServerCreation() {
	handler, err := mcpio.NewHandler(
		mcpio.WithName("database-server"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("read_record", "Get a record by ID", readRecord),
		mcpio.WithTool("list_records", "List all records with optional status filter", listRecords),
		mcpio.WithTool("create_record", "Create a new record with elicited data", createRecord),
		mcpio.WithTool("update_record", "Update a record with change confirmation", updateRecord),
		mcpio.WithTool("delete_record", "Delete a record with confirmation", deleteRecord),
	)
	s.Require().NoError(err)
	s.NotNil(handler)
	s.NotNil(handler.GetServer())
}

func (s *DatabaseTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
