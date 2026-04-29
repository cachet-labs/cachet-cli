package storage

import (
	"testing"
	"time"

	"github.com/cachet-labs/cachet-cli/internal/core"
)

func TestLocalStoreRoundtrip(t *testing.T) {
	dir := t.TempDir()
	store := NewLocalStore(dir)

	f := &core.Failure{
		ID:          "f_test-001",
		CapturedAt:  time.Now().UTC().Truncate(time.Second),
		Fingerprint: "POST:/pay:500:timeout",
		Request:     core.Request{Method: "POST", URL: "/pay"},
		Response:    core.Response{Status: 500},
		Error:       core.ErrorInfo{Type: "timeout", Message: "upstream timed out"},
	}

	if err := store.WriteFailure(f); err != nil {
		t.Fatal(err)
	}

	got, err := store.ReadFailure(f.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != f.ID {
		t.Errorf("ID: got %q, want %q", got.ID, f.ID)
	}
	if got.Fingerprint != f.Fingerprint {
		t.Errorf("Fingerprint: got %q, want %q", got.Fingerprint, f.Fingerprint)
	}
}

func TestLocalStoreNotFound(t *testing.T) {
	store := NewLocalStore(t.TempDir())
	_, err := store.ReadFailure("f_nonexistent")
	if err == nil {
		t.Error("expected error for missing failure")
	}
}
