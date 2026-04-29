package cmd

import (
	"fmt"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/llm"
	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/cachet-labs/cachet-cli/pkg/config"
	"github.com/spf13/cobra"
)

var askCmd = &cobra.Command{
	Use:   "ask <failure-id>",
	Short: "Diagnose a failure with AI",
	Long: `Build a structured prompt for the failure and send it to the configured LLM.

If no LLM is configured the prompt is printed to stdout (pipe-ready):
  cachet ask <id> | pbcopy`,
	Args: cobra.ExactArgs(1),
	RunE: runAsk,
}

func init() {
	rootCmd.AddCommand(askCmd)
}

func runAsk(cmd *cobra.Command, args []string) error {
	id := args[0]

	localStore := storage.NewLocalStore(".cachet/recent")
	failure, err := localStore.ReadFailure(id)
	if err != nil {
		return err
	}

	// Inject up to 3 similar cases from the global index.
	cases, err := loadSimilarCases(failure.Fingerprint)
	if err != nil {
		ui.Info(fmt.Sprintf("could not load similar cases: %v", err))
	}
	if ui.IsTTY() && len(cases) > 0 {
		ui.Info(fmt.Sprintf("found %d similar case(s) — injecting into prompt", len(cases)))
	}

	prompt := core.BuildPrompt(failure, cases)
	adapter := selectAdapter(cfg)

	switch adapter.(type) {
	case *llm.StdoutAdapter:
		if ui.IsTTY() {
			ui.Warn("No LLM configured — printing prompt (pipe-ready)")
			fmt.Println()
		}
	default:
		if ui.IsTTY() {
			ui.Info("Sending to " + cfg.Provider + " (" + cfg.Model + ")…")
			fmt.Println()
		}
	}

	response, err := adapter.Ask(prompt)
	if err != nil {
		return fmt.Errorf("ask: %w", err)
	}

	// StdoutAdapter already wrote to stdout.
	if response == "" {
		return nil
	}

	ui.DiagnosisBox(failure.Fingerprint, response)
	return nil
}

func loadSimilarCases(fingerprint string) ([]*core.Case, error) {
	idx, err := storage.NewIndex()
	if err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}
	ids := idx.Lookup(fingerprint)
	if len(ids) == 0 {
		return nil, nil
	}

	gs, err := storage.NewGlobalStore()
	if err != nil {
		return nil, fmt.Errorf("load global store: %w", err)
	}

	var cases []*core.Case
	for _, cid := range ids {
		if len(cases) >= 3 {
			break
		}
		c, err := gs.ReadCase(cid)
		if err != nil {
			continue
		}
		cases = append(cases, c)
	}
	return cases, nil
}

// selectAdapter returns the appropriate LLM adapter based on config.
func selectAdapter(cfg *config.Config) llm.Adapter {
	if cfg == nil {
		return &llm.StdoutAdapter{}
	}
	switch cfg.Provider {
	case "anthropic":
		if cfg.APIKey != "" {
			return llm.NewAnthropicAdapter(cfg.APIKey, cfg.Model)
		}
	}
	return &llm.StdoutAdapter{}
}
