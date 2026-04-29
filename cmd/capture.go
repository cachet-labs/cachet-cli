package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/pipeline"
	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	captureURL    string
	captureStatus int
	captureError  string
	captureBody   string
)

var captureCmd = &cobra.Command{
	Use:   "capture",
	Short: "Capture an API failure",
	Long: `Capture an API failure for later diagnosis.

Read from stdin (JSON):
  cat failure.json | cachet capture

Or use explicit flags:
  cachet capture --url POST:/pay --status 500 --error timeout`,
	RunE: runCapture,
}

func init() {
	captureCmd.Flags().StringVar(&captureURL, "url", "", "METHOD:PATH  (e.g. POST:/pay)")
	captureCmd.Flags().IntVar(&captureStatus, "status", 0, "HTTP status code")
	captureCmd.Flags().StringVar(&captureError, "error", "", "error type or message")
	captureCmd.Flags().StringVar(&captureBody, "body", "", "request body (optional)")
	rootCmd.AddCommand(captureCmd)
}

func runCapture(cmd *cobra.Command, args []string) error {
	var failure core.Failure

	stdinInfo, _ := os.Stdin.Stat()
	isStdinPipe := (stdinInfo.Mode() & os.ModeCharDevice) == 0

	if isStdinPipe {
		if err := json.NewDecoder(os.Stdin).Decode(&failure); err != nil {
			return fmt.Errorf("decode stdin JSON: %w", err)
		}
	} else {
		if captureURL == "" || captureStatus == 0 || captureError == "" {
			return fmt.Errorf("--url, --status, and --error are required when not piping JSON")
		}
		method, path := splitMethodPath(captureURL)
		failure = core.Failure{
			Request: core.Request{
				URL:    path,
				Method: method,
				Body:   captureBody,
			},
			Response: core.Response{
				Status: captureStatus,
			},
			Error: core.ErrorInfo{
				Type:    captureError,
				Message: captureError,
			},
		}
	}

	safe, err := pipeline.Ingest(&failure, cfg, storage.NewLocalStore(".cachet/recent"))
	if err != nil {
		return err
	}

	fmt.Println()
	ui.Success("Failure captured")
	fmt.Println()
	ui.KV(
		"ID", safe.ID,
		"Fingerprint", safe.Fingerprint,
		"Stored", ".cachet/recent/"+safe.ID+".json",
	)
	fmt.Println()
	ui.Hint("Run: cachet ask " + safe.ID + "  to diagnose")
	fmt.Println()

	return nil
}

// splitMethodPath splits "POST:/pay" → ("POST", "/pay").
// Handles absolute URLs: "POST:https://api.example.com/pay" → ("POST", "/pay").
func splitMethodPath(s string) (method, path string) {
	idx := strings.IndexByte(s, ':')
	if idx < 0 {
		return "GET", s
	}
	method = strings.ToUpper(s[:idx])
	rest := s[idx+1:]
	// If rest is a full or scheme-relative URL, extract only the path.
	if strings.Contains(rest, "://") || strings.HasPrefix(rest, "//") {
		if u, err := url.Parse(rest); err == nil && u.Path != "" {
			rest = u.Path
		}
	}
	return method, rest
}
