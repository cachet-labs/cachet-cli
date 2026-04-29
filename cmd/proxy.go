package cmd

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/proxy"
	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	proxyPort      int
	proxyTarget    string
	proxyMinStatus int
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Start a reverse proxy that auto-captures failing requests",
	Long: `Run a local reverse proxy that forwards all traffic to --target and
automatically captures any response at or above --min-status.

  cachet proxy --port 8080 --target http://localhost:3000
  cachet proxy --port 8080 --target http://localhost:3000 --min-status 500`,
	RunE: runProxy,
}

func init() {
	proxyCmd.Flags().IntVar(&proxyPort, "port", 8080, "port to listen on")
	proxyCmd.Flags().StringVar(&proxyTarget, "target", "", "upstream URL to proxy to (required)")
	proxyCmd.Flags().IntVar(&proxyMinStatus, "min-status", 400, "lowest HTTP status code to capture")
	_ = proxyCmd.MarkFlagRequired("target")
	rootCmd.AddCommand(proxyCmd)
}

func runProxy(cmd *cobra.Command, args []string) error {
	target, err := url.Parse(proxyTarget)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}
	if target.Scheme == "" || target.Host == "" {
		return fmt.Errorf("--target must be a full URL (e.g. http://localhost:3000)")
	}

	store := storage.NewLocalStore(".cachet/recent")
	p := proxy.New(cfg, store, proxyMinStatus, func(f *core.Failure) {
		fmt.Println()
		ui.Success(fmt.Sprintf("captured  %s %s  →  %d", f.Request.Method, f.Request.URL, f.Response.Status))
		ui.KV("ID", f.ID, "Fingerprint", f.Fingerprint)
		ui.Hint("cachet ask " + f.ID)
	})

	addr := fmt.Sprintf(":%d", proxyPort)
	ui.PrintBanner()
	ui.Info(fmt.Sprintf("Proxying :%d → %s", proxyPort, proxyTarget))
	ui.Info(fmt.Sprintf("Capturing status ≥ %d — press Ctrl+C to stop", proxyMinStatus))
	fmt.Println()

	return http.ListenAndServe(addr, p.Handler(target))
}
