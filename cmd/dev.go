package cmd

import (
	"fmt"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/devserver"
	"github.com/cachet-labs/cachet-cli/internal/ui"
	"github.com/cachet-labs/cachet-cli/pkg/config"
	"github.com/spf13/cobra"
)

var (
	devCommand   string
	devPort      int
	devProxyPort int
	devMinStatus int
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start dev server + capturing proxy together",
	Long: `Run your dev server and the cachet proxy as a single command.
Failures are captured automatically after the server handles its first
successful request — keeping boot-time noise out of .cachet/recent/.

Configure once in cachet.config.json:
  {
    "dev": {
      "command":   "bun run dev",
      "port":      3000,
      "proxyPort": 8080
    }
  }

Then replace your usual dev command with:
  cachet dev

Or pass flags directly (useful for first-time / one-off runs):
  cachet dev --command "npm run dev" --port 3000 --proxy-port 8080`,
	RunE: runDev,
}

func init() {
	devCmd.Flags().StringVar(&devCommand, "command", "", "dev server command (overrides config)")
	devCmd.Flags().IntVar(&devPort, "port", 0, "dev server port (overrides config)")
	devCmd.Flags().IntVar(&devProxyPort, "proxy-port", 0, "proxy port (overrides config, default 8080)")
	devCmd.Flags().IntVar(&devMinStatus, "min-status", 0, "lowest status code to capture (overrides config, default 400)")
	rootCmd.AddCommand(devCmd)
}

func runDev(cmd *cobra.Command, args []string) error {
	// Start from config defaults then apply CLI flag overrides.
	devCfg := config.DevConfig{
		Command:   "npm run dev",
		Port:      3000,
		ProxyPort: 8080,
		MinStatus: 400,
	}

	if cfg != nil {
		if cfg.Dev.Command != "" {
			devCfg.Command = cfg.Dev.Command
		}
		if cfg.Dev.Port != 0 {
			devCfg.Port = cfg.Dev.Port
		}
		if cfg.Dev.ProxyPort != 0 {
			devCfg.ProxyPort = cfg.Dev.ProxyPort
		}
		if cfg.Dev.MinStatus != 0 {
			devCfg.MinStatus = cfg.Dev.MinStatus
		}
	}

	if devCommand != "" {
		devCfg.Command = devCommand
	}
	if devPort != 0 {
		devCfg.Port = devPort
	}
	if devProxyPort != 0 {
		devCfg.ProxyPort = devProxyPort
	}
	if devMinStatus != 0 {
		devCfg.MinStatus = devMinStatus
	}

	if devCfg.Command == "" {
		return fmt.Errorf("no dev command configured — add a \"dev\" section to cachet.config.json or use --command")
	}

	ui.PrintBanner()
	ui.Info(fmt.Sprintf("Starting:  %s", devCfg.Command))
	ui.Info(fmt.Sprintf("Proxying:  :%d → :%d  (point your app clients at :%d)", devCfg.ProxyPort, devCfg.Port, devCfg.ProxyPort))
	ui.Info(fmt.Sprintf("Capturing: status ≥ %d  (held until first healthy response)", devCfg.MinStatus))
	fmt.Println()

	return devserver.Run(cfg, devCfg, func(f *core.Failure) {
		fmt.Println()
		ui.Success(fmt.Sprintf("captured  %s %s  →  %d", f.Request.Method, f.Request.URL, f.Response.Status))
		ui.KV("ID", f.ID, "Fingerprint", f.Fingerprint)
		ui.Hint("cachet ask " + f.ID)
	})
}
