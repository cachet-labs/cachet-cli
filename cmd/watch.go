package cmd

import (
	"fmt"
	"time"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/cachet-labs/cachet-cli/internal/watcher"
	"github.com/spf13/cobra"
)

var (
	watchNgrok     bool
	watchPort      int
	watchMinStatus int
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch a dev tunnel and auto-capture failing requests",
	Long: `Automatically capture failing API calls flowing through an active dev tunnel.

ngrok mode — polls the ngrok local inspection API every 2 seconds:
  cachet watch --ngrok
  cachet watch --ngrok --port 4041   (non-default ngrok inspection port)
  cachet watch --ngrok --min-status 500  (only 5xx)`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().BoolVar(&watchNgrok, "ngrok", false, "watch an ngrok tunnel (default inspection port: 4040)")
	watchCmd.Flags().IntVar(&watchPort, "port", 4040, "ngrok inspection API port")
	watchCmd.Flags().IntVar(&watchMinStatus, "min-status", 400, "lowest HTTP status code to capture")
	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
	if !watchNgrok {
		return fmt.Errorf("specify a tunnel source: --ngrok")
	}

	store := storage.NewLocalStore(".cachet/recent")
	base := fmt.Sprintf("http://localhost:%d", watchPort)

	onCapture := func(f *core.Failure) {
		fmt.Println()
		ui.Success(fmt.Sprintf("captured  %s %s  →  %d", f.Request.Method, f.Request.URL, f.Response.Status))
		ui.KV("ID", f.ID, "Fingerprint", f.Fingerprint)
		ui.Hint("cachet ask " + f.ID)
	}

	w := watcher.NewNgrokWatcher(base, cfg, store, watchMinStatus, onCapture)

	ui.PrintBanner()
	ui.Info(fmt.Sprintf("Watching ngrok at %s  (capturing status ≥ %d)", base, watchMinStatus))
	ui.Info("Press Ctrl+C to stop")
	fmt.Println()

	return w.Run(2 * time.Second)
}
