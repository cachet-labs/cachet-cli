package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cachet-labs/cachet-cli/internal/core"
)

// GlobalStore reads and writes Case records to ~/.cachet/cases/<id>.json.
type GlobalStore struct {
	dir string
}

// NewGlobalStore creates a GlobalStore rooted at ~/.cachet/cases.
func NewGlobalStore() (*GlobalStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	return &GlobalStore{dir: filepath.Join(home, ".cachet", "cases")}, nil
}

func (s *GlobalStore) ensureDir() error {
	return os.MkdirAll(s.dir, 0o755)
}

// WriteCase persists a resolved case.
func (s *GlobalStore) WriteCase(c *core.Case) error {
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("create global dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal case: %w", err)
	}
	path := filepath.Join(s.dir, c.ID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write case: %w", err)
	}
	return nil
}

// ReadCase loads a case by ID.
func (s *GlobalStore) ReadCase(id string) (*core.Case, error) {
	path := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("case %q not found", id)
		}
		return nil, fmt.Errorf("read case: %w", err)
	}
	var c core.Case
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse case: %w", err)
	}
	return &c, nil
}

// ListCases returns all cases sorted by filename (chronological by ID).
func (s *GlobalStore) ListCases() ([]*core.Case, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list cases: %w", err)
	}
	var cases []*core.Case
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		c, err := s.ReadCase(id)
		if err != nil {
			continue
		}
		cases = append(cases, c)
	}
	return cases, nil
}
