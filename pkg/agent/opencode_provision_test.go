package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ptone/scion-agent/pkg/config"
)

func TestProvisionOpencodeAgent(t *testing.T) {
	tmpDir := t.TempDir()

	// Move to tmpDir
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Mock HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Initialize a mock project
	projectDir := filepath.Join(tmpDir, "project")
	projectScionDir := filepath.Join(projectDir, ".scion")
	if err := config.InitProject(projectScionDir); err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	// Chdir to projectDir so GetProjectDir finds it
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}

	// Create dummy auth file
	authDir := filepath.Join(tmpDir, ".local", "share", "opencode")
	if err := os.MkdirAll(authDir, 0755); err != nil {
		t.Fatal(err)
	}
	authFile := filepath.Join(authDir, "auth.json")
	if err := os.WriteFile(authFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Provision an opencode agent
	agentName := "opencode-agent"
	_, _, _, err := ProvisionAgent(context.Background(), agentName, "opencode", "", projectScionDir, "", "", "", "")
	if err != nil {
		t.Fatalf("ProvisionAgent failed: %v", err)
	}

	// Verify agent's opencode.json
	agentOpencodeJSONPath := filepath.Join(projectScionDir, "agents", agentName, "home", ".config", "opencode.json")
	if _, err := os.Stat(agentOpencodeJSONPath); os.IsNotExist(err) {
		t.Fatalf("expected opencode.json to exist at %s", agentOpencodeJSONPath)
	}

	// Verify it has content
	data, err := os.ReadFile(agentOpencodeJSONPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("expected opencode.json to have content, but it's empty")
	}

	// Verify mountable dir and symlink
	mountableDir := filepath.Join(tmpDir, ".local", "share", "opencode-mountable")
	linkPath := filepath.Join(mountableDir, "auth.json")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("expected auth link to exist at %s: %v", linkPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected %s to be a symlink", linkPath)
	}
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	expectedTarget := filepath.Join(authDir, "auth.json")
	if target != expectedTarget {
		t.Errorf("expected link target %s, got %s", expectedTarget, target)
	}
}
