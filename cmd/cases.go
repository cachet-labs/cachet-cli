package cmd

import (
	"fmt"

	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/spf13/cobra"
)

var casesCmd = &cobra.Command{
	Use:   "cases",
	Short: "List all resolved cases",
	Long:  `List every resolved case stored in ~/.cachet/cases.`,
	RunE:  runCases,
}

func init() {
	rootCmd.AddCommand(casesCmd)
}

func runCases(cmd *cobra.Command, args []string) error {
	gs, err := storage.NewGlobalStore()
	if err != nil {
		return fmt.Errorf("load store: %w", err)
	}

	cases, err := gs.ListCases()
	if err != nil {
		return fmt.Errorf("list cases: %w", err)
	}

	if len(cases) == 0 {
		fmt.Println()
		ui.Hint("No cases yet.")
		ui.Hint("Run `cachet verify <failure-id>` after you fix a failure to store a case.")
		fmt.Println()
		return nil
	}

	ui.SectionHeader(fmt.Sprintf("Cases  (%d total)", len(cases)))

	headers := []string{"ID", "FINGERPRINT", "CATEGORY", "CONFIDENCE", "CREATED"}
	rows := make([][]string, len(cases))
	for i, c := range cases {
		rows[i] = []string{
			ui.ShortID(c.ID),
			c.Fingerprint,
			ui.CategoryBadge(c.Category),
			ui.ConfidencePct(c.Confidence),
			c.CreatedAt.Format("2006-01-02 15:04"),
		}
	}
	ui.Table(headers, rows)
	fmt.Println()

	return nil
}
