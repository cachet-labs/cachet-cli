package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/llm"
	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/cachet-labs/cachet-cli/pkg/config"
	"github.com/spf13/cobra"
)

// minCaseConfidence is the minimum confidence required for a past case to be
// injected into the ask prompt.
const minCaseConfidence = 0.5

var (
	askClipboard bool
	askLatest    bool
)

var askCmd = &cobra.Command{
	Use:   "ask [failure-id]",
	Short: "Diagnose a failure with AI",
	Long: `Build a structured prompt for the failure and send it to the configured LLM.

If no LLM is configured the prompt is printed to stdout (pipe-ready):
  cachet ask <id> | pbcopy
  cachet ask <id> --clipboard
  cachet ask --latest`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runAsk,
}

func init() {
	askCmd.Flags().BoolVar(&askClipboard, "clipboard", false, "copy response to clipboard")
	askCmd.Flags().BoolVar(&askLatest, "latest", false, "diagnose the most recently captured failure")
	rootCmd.AddCommand(askCmd)
}

func runAsk(cmd *cobra.Command, args []string) error {
	var id string
	if askLatest {
		s := storage.NewLocalStore(".cachet/recent")
		latest, err := s.LatestID()
		if err != nil {
			return err
		}
		if latest == "" {
			return fmt.Errorf("no failures captured yet")
		}
		id = latest
	} else {
		if len(args) == 0 {
			return fmt.Errorf("provide a failure ID or use --latest")
		}
		id = args[0]
	}

	localStore := storage.NewLocalStore(".cachet/recent")
	failure, err := localStore.ReadFailure(id)
	if err != nil {
		return err
	}

	// Inject up to 3 similar cases above the confidence threshold.
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

	// StdoutAdapter printed the prompt directly; nothing left to do.
	if _, isStdout := adapter.(*llm.StdoutAdapter); isStdout {
		return nil
	}

	ui.DiagnosisBox(failure.Fingerprint, response)

	if askClipboard {
		if err := copyToClipboard(response); err != nil {
			ui.Warn(fmt.Sprintf("clipboard copy failed: %v", err))
		} else if ui.IsTTY() {
			ui.Success("Response copied to clipboard")
		}
	}

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
		if c.Confidence >= minCaseConfidence {
			cases = append(cases, c)
		}
	}
	return cases, nil
}

// selectAdapter returns the appropriate LLM adapter based on config.
func selectAdapter(cfg *config.Config) llm.Adapter {
	if cfg == nil || cfg.APIKey == "" {
		return &llm.StdoutAdapter{}
	}
	switch cfg.Provider {
	case "anthropic":
		return llm.NewAnthropicAdapter(cfg.APIKey, cfg.Model)
	case "openai":
		return llm.NewOpenAIAdapter(cfg.APIKey, cfg.Model)
	}
	return &llm.StdoutAdapter{}
}

// copyToClipboard writes text to the system clipboard.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard utility found (install xclip or xsel)")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
