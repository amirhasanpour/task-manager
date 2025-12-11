package tests

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup code if needed
	
	// Run tests
	code := m.Run()
	
	// Teardown code if needed
	
	os.Exit(code)
}