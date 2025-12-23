package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ptone/scion/pkg/config"
	"github.com/ptone/scion/pkg/runtime"
	"github.com/spf13/cobra"
)

var (
	listAll bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List running scion agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		rt := runtime.GetRuntime()
		filters := map[string]string{
			"scion.agent": "true",
		}

		if !listAll {
			projectDir, err := config.GetResolvedProjectDir(grovePath)
			if err != nil {
				return err
			}
			groveName := config.GetGroveName(projectDir)
			filters["scion.grove"] = groveName
		}

		agents, err := rt.List(context.Background(), filters)
		if err != nil {
			return err
		}

		if len(agents) == 0 {
			if listAll {
				fmt.Println("No active agents found across any groves.")
			} else {
				fmt.Println("No active agents found in the current grove.")
			}
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tGROVE\tSTATUS\tID\tIMAGE")
		for _, a := range agents {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", a.Name, a.Grove, a.Status, a.ID, a.Image)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "List all agents across all groves")
}

