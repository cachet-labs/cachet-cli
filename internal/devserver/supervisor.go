package devserver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/proxy"
	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/pkg/config"
)

// Run starts the dev server command and a capturing proxy side-by-side.
// Captures are held until the dev server returns its first healthy response,
// preventing boot-time noise from a still-starting process.
func Run(cfg *config.Config, devCfg config.DevConfig, onCapture func(*core.Failure)) error {
	port := devCfg.Port
	if port == 0 {
		port = 3000
	}
	proxyPort := devCfg.ProxyPort
	if proxyPort == 0 {
		proxyPort = 8080
	}
	minStatus := devCfg.MinStatus
	if minStatus == 0 {
		minStatus = 400
	}

	target, err := url.Parse(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	store := storage.NewLocalStore(".cachet/recent")
	p := proxy.NewWithAutoArm(cfg, store, minStatus, onCapture)

	// Use sh -c for full shell semantics (env vars, pipes, etc.).
	devCmd := exec.Command("sh", "-c", devCfg.Command)
	devCmd.Stdout = os.Stdout
	devCmd.Stderr = os.Stderr
	devCmd.Stdin = os.Stdin

	if err := devCmd.Start(); err != nil {
		return fmt.Errorf("start dev server: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := fmt.Sprintf(":%d", proxyPort)
	srv := &http.Server{Addr: addr, Handler: p.Handler(target)}

	proxyErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			proxyErr <- err
		}
	}()

	devDone := make(chan error, 1)
	go func() {
		devDone <- devCmd.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = devCmd.Process.Kill()
		_ = srv.Close()
		return nil
	case err := <-proxyErr:
		_ = devCmd.Process.Kill()
		return fmt.Errorf("proxy: %w", err)
	case err := <-devDone:
		_ = srv.Close()
		if err != nil {
			return fmt.Errorf("dev server exited: %w", err)
		}
		return nil
	}
}
