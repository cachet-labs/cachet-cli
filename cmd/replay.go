package cmd

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/spf13/cobra"
)

var replayBaseURL string

var replayCmd = &cobra.Command{
	Use:   "replay <failure-id>",
	Short: "Re-execute a captured request and print the response",
	Long: `Re-send the HTTP request stored in a captured failure and print the live response.

If the stored URL is a relative path, supply --base-url:
  cachet replay <id> --base-url https://api.example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runReplay,
}

func init() {
	replayCmd.Flags().StringVar(&replayBaseURL, "base-url", "", "base URL prepended to relative paths")
	rootCmd.AddCommand(replayCmd)
}

func runReplay(cmd *cobra.Command, args []string) error {
	id := args[0]

	store := storage.NewLocalStore(".cachet/recent")
	failure, err := store.ReadFailure(id)
	if err != nil {
		return err
	}

	targetURL, err := resolveURL(failure.Request.URL, replayBaseURL)
	if err != nil {
		return err
	}

	if ui.IsTTY() {
		ui.Info(fmt.Sprintf("replaying  %s %s", failure.Request.Method, targetURL))
		fmt.Println()
	}

	status, body, err := doRequest(failure.Request.Method, targetURL, failure.Request.Headers, failure.Request.Body)
	if err != nil {
		return fmt.Errorf("replay request: %w", err)
	}

	if ui.IsTTY() {
		statusLabel := statusStyle(status)
		fmt.Printf("  %s %s\n\n", ui.Bold("Status"), statusLabel)
		ui.Box("Response Body", body)
	} else {
		fmt.Printf("Status: %d\n%s\n", status, body)
	}

	return nil
}

func resolveURL(rawURL, baseURL string) (string, error) {
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL, nil
	}
	if baseURL == "" {
		return "", fmt.Errorf("URL %q is a relative path — supply --base-url to replay", rawURL)
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(rawURL, "/"), nil
}

func doRequest(method, url string, headers map[string]string, body string) (int, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return 0, "", fmt.Errorf("build request: %w", err)
	}
	for k, v := range headers {
		// Skip redacted headers — they won't authenticate anyway.
		if v != "[REDACTED]" {
			req.Header.Set(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw), nil
}

func statusStyle(code int) string {
	s := fmt.Sprintf("%d", code)
	if !ui.IsTTY() {
		return s
	}
	switch {
	case code < 300:
		return ui.Bold(s) // green via caller
	case code < 400:
		return s
	default:
		return s
	}
}
