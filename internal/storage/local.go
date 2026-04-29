package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cachet-labs/cachet-cli/internal/core"
)

// LocalStore reads and writes Failure records to .cachet/recent/<id>.json.
type LocalStore struct {
	dir string
}

// NewLocalStore creates a LocalStore rooted at dir.
func NewLocalStore(dir string) *LocalStore {
	return &LocalStore{dir: dir}
}

func (s *LocalStore) ensureDir() error {
	return os.MkdirAll(s.dir, 0o755)
}

// WriteFailure persists f to disk. The caller must have already redacted f.
func (s *LocalStore) WriteFailure(f *core.Failure) error {
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("create local dir: %w", err)
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal failure: %w", err)
	}
	path := filepath.Join(s.dir, f.ID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write failure: %w", err)
	}
	return nil
}

// ReadFailure loads a failure by ID.
func (s *LocalStore) ReadFailure(id string) (*core.Failure, error) {
	path := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("failure %q not found in %s", id, s.dir)
		}
		return nil, fmt.Errorf("read failure: %w", err)
	}
	var f core.Failure
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse failure: %w", err)
	}
	return &f, nil
}

// ListIDs returns all failure IDs stored in the local directory.
func (s *LocalStore) ListIDs() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list failures: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			ids = append(ids, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return ids, nil
}
