package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/pipeline"
	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/pkg/config"
)

const maxBodyBytes = 1 << 20 // 1 MB cap to protect memory

// OnCapture is called (in a new goroutine) after a failure is stored.
type OnCapture func(f *core.Failure)

// Proxy is a transparent reverse proxy that captures failing HTTP responses.
type Proxy struct {
	cfg       *config.Config
	store     *storage.LocalStore
	minStatus int
	onCapture OnCapture
}

// New creates a Proxy that stores captured failures in store and invokes
// onCapture for every stored failure. minStatus is the lowest HTTP status code
// that triggers a capture (typically 400).
func New(cfg *config.Config, store *storage.LocalStore, minStatus int, onCapture OnCapture) *Proxy {
	return &Proxy{cfg: cfg, store: store, minStatus: minStatus, onCapture: onCapture}
}

// Handler returns an http.Handler that transparently proxies to target.
func (p *Proxy) Handler(target *url.URL) http.Handler {
	rp := httputil.NewSingleHostReverseProxy(target)
	rp.Transport = &captureTransport{base: http.DefaultTransport, proxy: p}

	// Capture connection-level errors (ECONNREFUSED, timeouts, etc.) as 502s.
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		method := r.Method
		path := r.URL.Path
		msg := err.Error()
		go p.capture(method, path, http.StatusBadGateway, nil, "", nil, "", core.ErrorInfo{
			Type:    "upstream_error",
			Message: msg,
		})
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "cachet proxy: upstream unreachable: %v\n", err)
	}
	return rp
}

type captureTransport struct {
	base  http.RoundTripper
	proxy *Proxy
}

func (t *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Buffer request body so it can be forwarded AND captured.
	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = io.ReadAll(io.LimitReader(req.Body, maxBodyBytes))
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
		req.ContentLength = int64(len(reqBody))
	}

	// Snapshot request fields before the goroutine runs to avoid data races.
	method := req.Method
	path := req.URL.Path
	reqHeaders := flattenHeaders(req.Header)

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= t.proxy.minStatus {
		var respBody []byte
		if resp.Body != nil {
			respBody, _ = io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
			// Drain and close the original body so the upstream TCP connection
			// is returned to the pool even when the response exceeds maxBodyBytes.
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
			resp.ContentLength = int64(len(respBody))
		}
		status := resp.StatusCode
		respHeaders := flattenHeaders(resp.Header)
		errInfo := errorInfoFromStatus(status, respBody)
		go t.proxy.capture(method, path, status, reqHeaders, string(reqBody), respHeaders, string(respBody), errInfo)
	}

	return resp, nil
}

func (p *Proxy) capture(
	method, path string,
	status int,
	reqHeaders map[string]string,
	reqBody string,
	respHeaders map[string]string,
	respBody string,
	errInfo core.ErrorInfo,
) {
	f := &core.Failure{
		Request: core.Request{
			Method:  method,
			URL:     path,
			Headers: reqHeaders,
			Body:    reqBody,
		},
		Response: core.Response{
			Status:  status,
			Headers: respHeaders,
			Body:    respBody,
		},
		Error: errInfo,
	}
	safe, err := pipeline.Ingest(f, p.cfg, p.store)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  cachet proxy: capture failed: %v\n", err)
		return
	}
	if p.onCapture != nil {
		p.onCapture(safe)
	}
}

func flattenHeaders(h http.Header) map[string]string {
	if h == nil {
		return nil
	}
	out := make(map[string]string, len(h))
	for k, vs := range h {
		out[k] = strings.Join(vs, ", ")
	}
	return out
}

func errorInfoFromStatus(status int, body []byte) core.ErrorInfo {
	errType := "server_error"
	if status >= 400 && status < 500 {
		errType = "client_error"
	}
	msg := http.StatusText(status)
	// Use the response body as the message when it's short enough to be readable.
	if len(body) > 0 && len(body) <= 512 {
		msg = strings.TrimSpace(string(body))
	}
	return core.ErrorInfo{Type: errType, Message: msg}
}
