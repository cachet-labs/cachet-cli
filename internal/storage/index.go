package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Index maps fingerprints to lists of case IDs in ~/.cachet/index.json.
type Index struct {
	path string
	data map[string][]string
}

// NewIndex loads the global index, creating it if absent.
func NewIndex() (*Index, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	path := filepath.Join(home, ".cachet", "index.json")
	idx := &Index{path: path, data: make(map[string][]string)}

	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read index: %w", err)
	}
	if err == nil {
		if err := json.Unmarshal(raw, &idx.data); err != nil {
			return nil, fmt.Errorf("parse index: %w", err)
		}
	}
	return idx, nil
}

// Add appends caseID under fingerprint and saves the index.
// It is a no-op if caseID is already present, preventing duplicate injections.
func (idx *Index) Add(fingerprint, caseID string) error {
	for _, id := range idx.data[fingerprint] {
		if id == caseID {
			return nil
		}
	}
	idx.data[fingerprint] = append(idx.data[fingerprint], caseID)
	return idx.save()
}

// Lookup returns case IDs for a fingerprint (nil if none).
func (idx *Index) Lookup(fingerprint string) []string {
	return idx.data[fingerprint]
}

func (idx *Index) save() error {
	if err := os.MkdirAll(filepath.Dir(idx.path), 0o755); err != nil {
		return fmt.Errorf("create index dir: %w", err)
	}
	data, err := json.MarshalIndent(idx.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	if err := os.WriteFile(idx.path, data, 0o644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	return nil
}
