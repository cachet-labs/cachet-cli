package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/llm"
	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	verifyNoReplay bool
	verifyBaseURL  string
	verifyDiffRef  string
)

var verifyCmd = &cobra.Command{
	Use:   "verify <failure-id>",
	Short: "Resolve a failure: replay + diff → AI extracts root cause → stores Case",
	Long: `Mark a failure as resolved by capturing the git diff and optionally replaying
the request. The LLM extracts a structured Case (root cause, fix, category,
confidence) which is stored globally and indexed for future `+"`cachet ask`"+` calls.

  cachet verify <id>                  # replay + git diff HEAD~1
  cachet verify <id> --no-replay      # diff only (safe for non-idempotent endpoints)
  cachet verify <id> --diff HEAD~2    # custom git base ref
  cachet verify <id> --base-url https://api.example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runVerify,
}

func init() {
	verifyCmd.Flags().BoolVar(&verifyNoReplay, "no-replay", false, "skip request replay, use diff only")
	verifyCmd.Flags().StringVar(&verifyBaseURL, "base-url", "", "base URL for replaying relative paths")
	verifyCmd.Flags().StringVar(&verifyDiffRef, "diff", "HEAD~1", "git base ref for the fix diff")
	rootCmd.AddCommand(verifyCmd)
}

func runVerify(cmd *cobra.Command, args []string) error {
	id := args[0]

	localStore := storage.NewLocalStore(".cachet/recent")
	failure, err := localStore.ReadFailure(id)
	if err != nil {
		return err
	}

	// ── Step 1: optional replay ───────────────────────────────────────────────
	if !verifyNoReplay {
		targetURL, err := resolveURL(failure.Request.URL, verifyBaseURL)
		if err != nil {
			ui.Warn(fmt.Sprintf("skipping replay: %v", err))
		} else {
			if ui.IsTTY() {
				ui.Info(fmt.Sprintf("replaying  %s %s", failure.Request.Method, targetURL))
			}
			status, _, replayErr := doRequest(
				failure.Request.Method, targetURL,
				failure.Request.Headers, failure.Request.Body,
			)
			if replayErr != nil {
				ui.Warn(fmt.Sprintf("replay failed: %v — continuing with diff only", replayErr))
			} else if status >= 500 {
				ui.Warn(fmt.Sprintf("replay returned %d — the fix may not be deployed yet", status))
			} else {
				if ui.IsTTY() {
					ui.Success(fmt.Sprintf("replay returned %d — endpoint looks healthy", status))
				}
			}
		}
	}

	// ── Step 2: git diff ─────────────────────────────────────────────────────
	diff, err := gitDiff(verifyDiffRef)
	if err != nil {
		ui.Warn(fmt.Sprintf("could not get git diff: %v — continuing without it", err))
		diff = ""
	} else if ui.IsTTY() {
		lines := len(strings.Split(strings.TrimSpace(diff), "\n"))
		ui.Info(fmt.Sprintf("captured git diff %s (%d lines)", verifyDiffRef, lines))
	}

	// ── Step 3: resolver prompt ───────────────────────────────────────────────
	adapter := selectAdapter(cfg)
	if _, ok := adapter.(*llm.StdoutAdapter); ok {
		// verify requires a real LLM to produce a structured Case.
		return fmt.Errorf("verify requires an LLM — set provider and apiKey in cachet.config.json")
	}

	prompt := core.BuildResolverPrompt(failure, diff)

	if ui.IsTTY() {
		ui.Info(fmt.Sprintf("sending resolver prompt to %s…", cfg.Provider))
		fmt.Println()
	}

	response, err := adapter.Ask(prompt)
	if err != nil {
		return fmt.Errorf("resolver LLM call: %w", err)
	}

	// ── Step 4: parse + store ─────────────────────────────────────────────────
	resolvedCase, err := core.ParseResolverResponse(response, failure.Fingerprint)
	if err != nil {
		return fmt.Errorf("parse resolver response: %w\n\nRaw response:\n%s", err, response)
	}

	gs, err := storage.NewGlobalStore()
	if err != nil {
		return fmt.Errorf("open global store: %w", err)
	}
	if err := gs.WriteCase(resolvedCase); err != nil {
		return fmt.Errorf("store case: %w", err)
	}

	idx, err := storage.NewIndex()
	if err != nil {
		return fmt.Errorf("open index: %w", err)
	}
	if err := idx.Add(resolvedCase.Fingerprint, resolvedCase.ID); err != nil {
		return fmt.Errorf("update index: %w", err)
	}

	// ── Step 5: output ────────────────────────────────────────────────────────
	fmt.Println()
	ui.Success("Case resolved and stored")
	fmt.Println()
	ui.KV(
		"Case ID", resolvedCase.ID,
		"Fingerprint", resolvedCase.Fingerprint,
		"Category", ui.CategoryBadge(resolvedCase.Category),
		"Confidence", ui.ConfidencePct(resolvedCase.Confidence),
	)
	fmt.Println()
	ui.Hint("Root Cause: " + resolvedCase.RootCause)
	ui.Hint("Fix:        " + resolvedCase.Fix)
	fmt.Println()
	ui.Hint("This case will now appear in future `cachet ask` prompts for " + resolvedCase.Fingerprint)
	fmt.Println()

	return nil
}

// gitDiff runs `git diff <baseRef>` and returns the output.
func gitDiff(baseRef string) (string, error) {
	out, err := exec.Command("git", "diff", baseRef).Output()
	if err != nil {
		return "", fmt.Errorf("git diff %s: %w", baseRef, err)
	}
	return string(out), nil
}

