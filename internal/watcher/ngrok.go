package watcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/pipeline"
	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/pkg/config"
)

const ngrokAPIPath = "/api/requests/http"

// ngrok local inspection API response shapes.
type ngrokResponse struct {
	Requests []ngrokItem `json:"requests"`
}

type ngrokItem struct {
	ID       string        `json:"id"`
	Request  ngrokReqData  `json:"request"`
	Response ngrokRespData `json:"response"`
}

type ngrokReqData struct {
	Method  string              `json:"method"`
	URI     string              `json:"uri"`
	Headers map[string][]string `json:"headers"`
}

type ngrokRespData struct {
	StatusCode int                 `json:"status_code"`
	Status     string              `json:"status"`
	Headers    map[string][]string `json:"headers"`
}

// NgrokWatcher polls the ngrok local inspection API and captures failing requests.
type NgrokWatcher struct {
	apiBase   string
	cfg       *config.Config
	store     *storage.LocalStore
	minStatus int
	seen      map[string]bool
	onCapture func(*core.Failure)
	client    *http.Client
}

// NewNgrokWatcher creates a watcher targeting apiBase (e.g. "http://localhost:4040").
func NewNgrokWatcher(apiBase string, cfg *config.Config, store *storage.LocalStore, minStatus int, onCapture func(*core.Failure)) *NgrokWatcher {
	return &NgrokWatcher{
		apiBase:   strings.TrimRight(apiBase, "/"),
		cfg:       cfg,
		store:     store,
		minStatus: minStatus,
		seen:      make(map[string]bool),
		onCapture: onCapture,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Run polls ngrok every interval until the process exits. Returns an error only
// if the initial reachability check fails.
func (w *NgrokWatcher) Run(interval time.Duration) error {
	if err := w.ping(); err != nil {
		return fmt.Errorf("cannot reach ngrok inspection API at %s%s: %w\n  Is ngrok running? (ngrok http <port>)", w.apiBase, ngrokAPIPath, err)
	}
	for {
		if err := w.poll(); err != nil {
			// Non-fatal — ngrok may have briefly restarted.
			fmt.Printf("  poll error: %v\n", err)
		}
		time.Sleep(interval)
	}
}

func (w *NgrokWatcher) ping() error {
	resp, err := w.client.Get(w.apiBase + ngrokAPIPath)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (w *NgrokWatcher) poll() error {
	resp, err := w.client.Get(w.apiBase + ngrokAPIPath)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var data ngrokResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("decode ngrok response: %w", err)
	}

	for _, item := range data.Requests {
		if w.seen[item.ID] {
			continue
		}
		w.seen[item.ID] = true

		if item.Response.StatusCode < w.minStatus {
			continue
		}

		f := toFailure(item)
		safe, err := pipeline.Ingest(f, w.cfg, w.store)
		if err != nil {
			continue
		}
		if w.onCapture != nil {
			w.onCapture(safe)
		}
	}
	return nil
}

func toFailure(item ngrokItem) *core.Failure {
	// Extract the path portion from the full URI for fingerprinting.
	path := item.Request.URI
	if idx := strings.Index(path, "://"); idx >= 0 {
		rest := path[idx+3:]
		if slash := strings.Index(rest, "/"); slash >= 0 {
			path = rest[slash:]
		} else {
			path = "/"
		}
	}

	errType := "server_error"
	if item.Response.StatusCode >= 400 && item.Response.StatusCode < 500 {
		errType = "client_error"
	}

	return &core.Failure{
		Request: core.Request{
			Method:  item.Request.Method,
			URL:     path,
			Headers: flattenHeaders(item.Request.Headers),
		},
		Response: core.Response{
			Status:  item.Response.StatusCode,
			Headers: flattenHeaders(item.Response.Headers),
		},
		Error: core.ErrorInfo{
			Type:    errType,
			Message: strings.TrimPrefix(item.Response.Status, fmt.Sprintf("%d ", item.Response.StatusCode)),
		},
	}
}

func flattenHeaders(h map[string][]string) map[string]string {
	if h == nil {
		return nil
	}
	out := make(map[string]string, len(h))
	for k, vs := range h {
		out[k] = strings.Join(vs, ", ")
	}
	return out
}
