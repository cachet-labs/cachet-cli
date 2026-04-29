package cmd

import (
	"fmt"
	"strings"

	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <case-id>",
	Short: "Inspect a resolved case",
	Long:  `Print all fields of a stored case.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	id := args[0]

	gs, err := storage.NewGlobalStore()
	if err != nil {
		return fmt.Errorf("load store: %w", err)
	}
	c, err := gs.ReadCase(id)
	if err != nil {
		return err
	}

	var sb strings.Builder

	// Metadata block.
	meta := []struct{ k, v string }{
		{"ID", c.ID},
		{"Fingerprint", c.Fingerprint},
		{"Category", ui.CategoryBadge(c.Category)},
		{"Confidence", ui.ConfidencePct(c.Confidence)},
		{"Created", c.CreatedAt.Format("2006-01-02 15:04:05 UTC")},
	}
	maxKey := 0
	for _, m := range meta {
		if len(m.k) > maxKey {
			maxKey = len(m.k)
		}
	}
	for _, m := range meta {
		pad := strings.Repeat(" ", maxKey-len(m.k))
		fmt.Fprintf(&sb, "  %s%s   %s\n", ui.Bold(m.k), pad, m.v)
	}

	// Root cause and fix sections.
	fmt.Fprintf(&sb, "\n  %s\n  %s\n", ui.Bold("Root Cause"), c.RootCause)
	fmt.Fprintf(&sb, "\n  %s\n  %s\n", ui.Bold("Fix"), c.Fix)

	ui.Box("Case  "+ui.ShortID(c.ID), sb.String())
	return nil
}
