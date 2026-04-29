package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/spf13/cobra"
)

var casesFilter string

var casesCmd = &cobra.Command{
	Use:   "cases",
	Short: "List all resolved cases",
	Long: `List every resolved case stored in ~/.cachet/cases.

Filter examples:
  cachet cases --filter category=timeout
  cachet cases --filter confidence=0.8`,
	RunE: runCases,
}

func init() {
	casesCmd.Flags().StringVar(&casesFilter, "filter", "", "filter by field (category=<val> or confidence=<min>)")
	rootCmd.AddCommand(casesCmd)
}

func runCases(cmd *cobra.Command, args []string) error {
	gs, err := storage.NewGlobalStore()
	if err != nil {
		return fmt.Errorf("load store: %w", err)
	}

	all, err := gs.ListCases()
	if err != nil {
		return fmt.Errorf("list cases: %w", err)
	}

	cases, err := applyFilter(all, casesFilter)
	if err != nil {
		return fmt.Errorf("invalid filter: %w", err)
	}

	if len(cases) == 0 {
		fmt.Println()
		if casesFilter != "" {
			ui.Hint(fmt.Sprintf("No cases match filter %q.", casesFilter))
		} else {
			ui.Hint("No cases yet.")
			ui.Hint("Run `cachet verify <failure-id>` after you fix a failure to store a case.")
		}
		fmt.Println()
		return nil
	}

	label := fmt.Sprintf("Cases  (%d total)", len(cases))
	if casesFilter != "" {
		label = fmt.Sprintf("Cases  (%d match: %s)", len(cases), casesFilter)
	}
	ui.SectionHeader(label)

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

// applyFilter returns the subset of cases matching the filter string.
// Supported filters: category=<value>, confidence=<min-float>.
// An empty filter returns all cases unchanged.
func applyFilter(cases []*core.Case, filter string) ([]*core.Case, error) {
	if filter == "" {
		return cases, nil
	}

	key, val, ok := strings.Cut(filter, "=")
	if !ok {
		return nil, fmt.Errorf("expected key=value, got %q", filter)
	}
	key = strings.TrimSpace(key)
	val = strings.TrimSpace(val)

	var out []*core.Case
	switch strings.ToLower(key) {
	case "category":
		for _, c := range cases {
			if strings.EqualFold(c.Category, val) {
				out = append(out, c)
			}
		}
	case "confidence":
		min, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("confidence must be a float, got %q", val)
		}
		for _, c := range cases {
			if c.Confidence >= min {
				out = append(out, c)
			}
		}
	default:
		return nil, fmt.Errorf("unknown filter key %q (supported: category, confidence)", key)
	}
	return out, nil
}
