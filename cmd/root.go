package cmd

import (
	"os"

	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/cachet-labs/cachet-cli/pkg/config"
	"github.com/spf13/cobra"
)

var (
	cfg     *config.Config
	version string
)

var rootCmd = &cobra.Command{
	Use:   "cachet",
	Short: "Turn API failures into structured AI-debugging context",
	Long: `◆ cachet — runtime failure intelligence

Capture API failures, ask for AI diagnosis, and build a persistent
knowledge base of root causes and fixes.

  cachet capture --url POST:/pay --status 500 --error timeout
  cachet ask <failure-id>
  cachet cases
  cachet show <case-id>`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute is the CLI entrypoint called from main.
func Execute(v string) {
	version = v
	rootCmd.Version = v
	if err := rootCmd.Execute(); err != nil {
		ui.Error(err.Error())
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		ui.Error("config: " + err.Error())
		os.Exit(1)
	}
}
