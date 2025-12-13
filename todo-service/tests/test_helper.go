package tests

import (
	"os"
	"testing"
)

// TestMain handles global test setup/teardown
func TestMain(m *testing.M) {
	// Setup code if needed
	setup()
	
	// Run tests
	code := m.Run()
	
	// Teardown code if needed
	teardown()
	
	os.Exit(code)
}

func setup() {
	// Global test setup
}

func teardown() {
	// Global test teardown
}

// SkipIfShort skips the test in short mode
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// RunIfIntegration runs only if integration flag is set
func RunIfIntegration(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}
}