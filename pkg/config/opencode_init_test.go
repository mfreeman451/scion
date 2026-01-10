package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpencodeTemplateSeeding(t *testing.T) {
	// Setup a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "scion-opencode-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home dir for global templates
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create a mock project structure
	projectDir := filepath.Join(tmpDir, "project", DotScion)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Helper to change current working directory
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	if err := os.Chdir(filepath.Dir(projectDir)); err != nil {
		t.Fatal(err)
	}

	// Test initializing project (should now include opencode)
	err = InitProject("")
	if err != nil {
		t.Fatalf("failed to init project: %v", err)
	}

	opencodePath := filepath.Join(projectDir, "templates", "opencode")
	if _, err := os.Stat(opencodePath); os.IsNotExist(err) {
		t.Errorf("expected opencode template directory %s to exist", opencodePath)
	}

	// Verify opencode.json exists in the correct location
	opencodeJSONPath := filepath.Join(opencodePath, "home", ".config", "opencode", "opencode.json")
	if _, err := os.Stat(opencodeJSONPath); os.IsNotExist(err) {
		t.Errorf("expected opencode.json to exist at %s", opencodeJSONPath)
	}

	// Verify gemini-specific material is NOT there
	opencodeConfigPath := filepath.Join(opencodePath, "home", ".opencode")
	if _, err := os.Stat(opencodeConfigPath); err == nil {
		t.Error("expected .opencode directory to NOT exist in opencode template")
	}

	// Verify it's not empty
	data, err := os.ReadFile(opencodeJSONPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("expected opencode.json to have content, but it's empty")
	}

	// Verify protected from deletion
	err = DeleteTemplate("opencode", false)
	if err == nil {
		t.Error("expected error when deleting opencode template, got nil")
	}
}
