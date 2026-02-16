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

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/ptone/scion-agent/pkg/config"
	"github.com/ptone/scion-agent/pkg/harness"
	"github.com/spf13/cobra"
)

var harnessConfigCmd = &cobra.Command{
	Use:     "harness-config",
	Aliases: []string{"hc"},
	Short:   "Manage harness configurations",
	Long:    `List and manage harness-config directories that define runtime settings for each harness type.`,
}

var harnessConfigListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available harness configurations",
	RunE: func(cmd *cobra.Command, args []string) error {
		var gp string
		if grovePath != "" {
			resolved, err := config.GetResolvedProjectDir(grovePath)
			if err == nil {
				gp = resolved
			}
		} else if projectDir, err := config.GetResolvedProjectDir(""); err == nil {
			gp = projectDir
		}

		configs, err := config.ListHarnessConfigDirs(gp)
		if err != nil {
			return fmt.Errorf("failed to list harness configs: %w", err)
		}

		if len(configs) == 0 {
			fmt.Println("No harness configurations found.")
			fmt.Println("Run 'scion init --machine' to seed default harness configurations.")
			return nil
		}

		if isJSONOutput() {
			type hcEntry struct {
				Name    string `json:"name"`
				Harness string `json:"harness"`
				Image   string `json:"image,omitempty"`
				Path    string `json:"path"`
			}
			entries := make([]hcEntry, 0, len(configs))
			for _, hc := range configs {
				entries = append(entries, hcEntry{
					Name:    hc.Name,
					Harness: hc.Config.Harness,
					Image:   hc.Config.Image,
					Path:    hc.Path,
				})
			}
			return outputJSON(entries)
		}

		fmt.Printf("%-20s %-12s %s\n", "NAME", "HARNESS", "IMAGE")
		for _, hc := range configs {
			image := hc.Config.Image
			if len(image) > 60 {
				image = "..." + image[len(image)-57:]
			}
			fmt.Printf("%-20s %-12s %s\n", hc.Name, hc.Config.Harness, image)
		}
		return nil
	},
}

var harnessConfigResetCmd = &cobra.Command{
	Use:   "reset <name>",
	Short: "Reset a harness configuration to its embedded defaults",
	Long: `Restores a harness-config directory to the embedded defaults.
This overwrites config.yaml and home directory files with the built-in versions.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Resolve the target directory (always global since that's where harness-configs live)
		globalDir, err := config.GetGlobalDir()
		if err != nil {
			return fmt.Errorf("failed to resolve global directory: %w", err)
		}

		targetDir := filepath.Join(globalDir, "harness-configs", name)

		// Load existing config to determine harness type
		hcDir, err := config.LoadHarnessConfigDir(targetDir)
		if err != nil {
			return fmt.Errorf("harness-config %q not found at %s: %w", name, targetDir, err)
		}

		// Find the matching harness implementation
		h := harness.New(hcDir.Config.Harness)

		// Reset by re-seeding with force=true
		if err := config.SeedHarnessConfig(targetDir, h, true); err != nil {
			return fmt.Errorf("failed to reset harness-config %q: %w", name, err)
		}

		if isJSONOutput() {
			return outputJSON(ActionResult{
				Status:  "success",
				Command: "harness-config reset",
				Message: fmt.Sprintf("Harness-config %q reset to defaults.", name),
				Details: map[string]interface{}{
					"name":    name,
					"harness": hcDir.Config.Harness,
				},
			})
		}

		fmt.Printf("Harness-config %q reset to defaults.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(harnessConfigCmd)
	harnessConfigCmd.AddCommand(harnessConfigListCmd)
	harnessConfigCmd.AddCommand(harnessConfigResetCmd)
}
