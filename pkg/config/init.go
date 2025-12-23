package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed embeds/scion_hook.py
var DefaultScionHookPy string

//go:embed embeds/settings.json
var DefaultSettingsJSON string

//go:embed embeds/system_prompt.md
var DefaultSystemPrompt string

//go:embed embeds/scion.json
var DefaultScionJSON string

//go:embed embeds/gemini.md
var DefaultGeminiMD string

//go:embed embeds/bashrc
var DefaultBashrc string

func SeedTemplateDir(templateDir string, templateName string) error {
	// Create directories
	dirs := []string{
		templateDir,
		filepath.Join(templateDir, ".gemini"),
		filepath.Join(templateDir, ".config", "gcloud"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	scionJSON := DefaultScionJSON
	if templateName != "" && templateName != "default" {
		scionJSON = strings.Replace(scionJSON, `"template": "default"`, fmt.Sprintf(`"template": %q`, templateName), 1)
	}

	// Seed template files
	files := []struct {
		path    string
		content string
	}{
		{filepath.Join(templateDir, "scion.json"), scionJSON},
		{filepath.Join(templateDir, "scion_hook.py"), DefaultScionHookPy},
		{filepath.Join(templateDir, ".gemini", "settings.json"), DefaultSettingsJSON},
		{filepath.Join(templateDir, ".gemini", "system_prompt.md"), DefaultSystemPrompt},
		{filepath.Join(templateDir, ".gemini", "gemini.md"), DefaultGeminiMD},
		{filepath.Join(templateDir, ".bashrc"), DefaultBashrc},
	}

	for _, f := range files {
		// Always write settings.json to ensure it matches current defaults
		if filepath.Base(f.path) == "settings.json" {
			if err := os.WriteFile(f.path, []byte(f.content), 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", f.path, err)
			}
			continue
		}

		if _, err := os.Stat(f.path); os.IsNotExist(err) {
			if err := os.WriteFile(f.path, []byte(f.content), 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", f.path, err)
			}
		}
	}

	return nil
}

func InitProject(targetDir string) error {
	var projectDir string
	var err error

	if targetDir != "" {
		projectDir = targetDir
	} else {
		projectDir, err = GetTargetProjectDir()
		if err != nil {
			return err
		}
	}

	templatesDir := filepath.Join(projectDir, "templates")
	defaultTemplateDir := filepath.Join(templatesDir, "default")
	agentsDir := filepath.Join(projectDir, "agents")

	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	return SeedTemplateDir(defaultTemplateDir, "default")
}

func InitGlobal() error {
	globalDir, err := GetGlobalDir()
	if err != nil {
		return err
	}

	templatesDir := filepath.Join(globalDir, "templates")
	defaultTemplateDir := filepath.Join(templatesDir, "default")
	agentsDir := filepath.Join(globalDir, "agents")

	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create global agents directory: %w", err)
	}

	return SeedTemplateDir(defaultTemplateDir, "default")
}
