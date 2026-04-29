package cmd

import (
	"fmt"

	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/spf13/cobra"
)

var latestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Print the ID of the most recently captured failure",
	Long: `Print the failure ID to stdout — useful in shell pipelines:

  cachet ask $(cachet latest)
  cachet show $(cachet latest)`,
	RunE: runLatest,
}

func init() {
	rootCmd.AddCommand(latestCmd)
}

func runLatest(cmd *cobra.Command, args []string) error {
	store := storage.NewLocalStore(".cachet/recent")
	id, err := store.LatestID()
	if err != nil {
		return err
	}
	if id == "" {
		ui.Warn("No failures captured yet")
		return nil
	}
	fmt.Println(id)
	return nil
}
