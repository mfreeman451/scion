package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTemplate(t *testing.T) {
	// Setup a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "scion-test-*")
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

	// Test creating a project template
	tplName := "test-tpl"
	err = CreateTemplate(tplName, false)
	if err != nil {
		t.Fatalf("failed to create project template: %v", err)
	}

	expectedPath := filepath.Join(projectDir, "templates", tplName)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected template directory %s to exist", expectedPath)
	}

	// Verify key files exist
	files := []string{
		"scion.json",
		".bashrc",
		filepath.Join(".gemini", "settings.json"),
	}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(expectedPath, f)); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist in template", f)
		}
	}

	// Test creating a global template
	globalTplName := "global-tpl"
	err = CreateTemplate(globalTplName, true)
	if err != nil {
		t.Fatalf("failed to create global template: %v", err)
	}

	globalExpectedPath := filepath.Join(tmpDir, GlobalDir, "templates", globalTplName)
	if _, err := os.Stat(globalExpectedPath); os.IsNotExist(err) {
		t.Errorf("expected global template directory %s to exist", globalExpectedPath)
	}

	// Test duplicate template creation fails
	err = CreateTemplate(tplName, false)
	if err == nil {
		t.Error("expected error when creating duplicate template, got nil")
	}
}
