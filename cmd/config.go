package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage cachet configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactively create cachet.config.json",
	Long: `Walk through a short setup wizard and write cachet.config.json
in the current directory. The file is gitignored by default.

Environment variable overrides are always respected even without this file:
  CACHET_PROVIDER, CACHET_API_KEY, CACHET_MODEL`,
	RunE: runConfigInit,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigInit(_ *cobra.Command, _ []string) error {
	const configFile = "cachet.config.json"

	if _, err := os.Stat(configFile); err == nil {
		ui.Warn(configFile + " already exists — overwrite? [y/N] ")
		var ans string
		fmt.Scanln(&ans) //nolint:errcheck
		if strings.ToLower(strings.TrimSpace(ans)) != "y" {
			ui.Hint("Aborted.")
			return nil
		}
	}

	r := bufio.NewReader(os.Stdin)

	fmt.Println()
	ui.PrintBanner()

	// Provider
	provider := prompt(r, "Provider", "anthropic", []string{"anthropic", "openai"})

	// API key
	fmt.Printf("  %s  ", ui.Bold("API Key"))
	apiKey, _ := r.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		ui.Warn("No API key entered. You can set CACHET_API_KEY instead.")
	}

	// Model
	defaultModel := "claude-sonnet-4-6"
	if provider == "openai" {
		defaultModel = "gpt-4o"
	}
	model := prompt(r, "Model", defaultModel, nil)

	// Temperature
	tempStr := prompt(r, "Temperature", "0.2", nil)
	temp := 0.2
	fmt.Sscanf(tempStr, "%f", &temp) //nolint:errcheck

	fmt.Println()

	out := map[string]any{
		"provider":    provider,
		"apiKey":      apiKey,
		"model":       model,
		"temperature": temp,
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(configFile, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	ui.Success("Saved " + configFile)
	fmt.Println()
	ui.Hint("Test it:  cachet ask <failure-id>")
	ui.Hint("Env override:  CACHET_API_KEY overrides the stored key at any time.")
	fmt.Println()

	return nil
}

// prompt prints a labelled prompt with a default value and reads one line.
func prompt(r *bufio.Reader, label, def string, options []string) string {
	var hint string
	if len(options) > 0 {
		hint = fmt.Sprintf(" [%s]", strings.Join(options, "/"))
	}
	if def != "" {
		hint += fmt.Sprintf(" (default: %s)", def)
	}
	fmt.Printf("  %s%s: ", ui.Bold(label), hint)

	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}
