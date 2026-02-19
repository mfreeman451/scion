// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitProject_CreatesEmptyTemplatesDir(t *testing.T) {
	// Create a temporary directory for the project
	tempDir, err := os.MkdirTemp("", "scion-init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Run InitProject
	err = InitProject(tempDir, GetMockHarnesses())
	if err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	// Verify that templates/ directory exists
	templatesDir := filepath.Join(tempDir, "templates")
	if info, err := os.Stat(templatesDir); err != nil || !info.IsDir() {
		t.Fatalf("Expected templates/ directory to exist at %s", templatesDir)
	}

	// Verify that templates/default does NOT exist (default template lives in global grove only)
	defaultDir := filepath.Join(tempDir, "templates", "default")
	if _, err := os.Stat(defaultDir); !os.IsNotExist(err) {
		t.Errorf("Expected templates/default to NOT exist at project level, but it does at %s", defaultDir)
	}

	// Verify per-harness templates were NOT created
	for _, name := range []string{"gemini", "claude", "opencode", "codex"} {
		perHarnessDir := filepath.Join(tempDir, "templates", name)
		if _, err := os.Stat(perHarnessDir); !os.IsNotExist(err) {
			t.Errorf("Expected per-harness template %s to NOT be created at project level", name)
		}
	}
}
