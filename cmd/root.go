/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)



var (
	grovePath string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "scion",
	Short: "A container-based orchestration tool for managing concurrent Gemini CLI agents",
	Long: `Scion is a container-based orchestration tool for managing 
concurrent Gemini CLI agents. It enables parallel execution of specialized 
sub-agents with isolated identities, credentials, and workspaces.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&grovePath, "grove", "g", "", "Path to a .scion grove directory")
}


