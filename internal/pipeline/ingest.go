package pipeline

import (
	"fmt"
	"time"

	"github.com/cachet-labs/cachet-cli/internal/core"
	"github.com/cachet-labs/cachet-cli/internal/storage"
	"github.com/cachet-labs/cachet-cli/pkg/config"
	"github.com/google/uuid"
)

// Ingest applies the mandatory redact → fingerprint → store sequence to f and
// returns the sanitised copy that was written to disk. Callers must never write
// f to disk directly — this function is the single enforcement point for
// invariant #1 (redaction is always first).
func Ingest(f *core.Failure, cfg *config.Config, store *storage.LocalStore) (*core.Failure, error) {
	if f.ID == "" {
		f.ID = "f_" + uuid.New().String()
	}
	if f.CapturedAt.IsZero() {
		f.CapturedAt = time.Now().UTC()
	}

	redactor, err := core.NewRedactor(cfg.Redact.Headers, cfg.Redact.Patterns)
	if err != nil {
		return nil, fmt.Errorf("create redactor: %w", err)
	}
	safe := redactor.RedactFailure(f)
	safe.Fingerprint = core.Fingerprint(safe.Request.Method, safe.Request.URL, safe.Response.Status, safe.Error.Type)

	if err := store.WriteFailure(safe); err != nil {
		return nil, fmt.Errorf("store failure: %w", err)
	}
	return safe, nil
}
