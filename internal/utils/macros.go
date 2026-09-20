package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DailyMacros is the user's daily macronutrient target, in grams.
type DailyMacros struct {
	Proteins float64 `json:"proteins"`
	Fats     float64 `json:"fats"`
	Carbs    float64 `json:"carbs"`
}

// MacroStore holds the daily macro target and persists it to a JSON file.
// It is safe for concurrent use.
type MacroStore struct {
	path string
	mu   sync.RWMutex
	cur  DailyMacros
}

// NewMacroStore loads the target from path. A missing file yields a zero-value
// target (and no file is created until the first Set); any other read or parse
// error is returned.
func NewMacroStore(path string) (*MacroStore, error) {
	s := &MacroStore{path: path}
	raw, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return s, nil
	case err != nil:
		return nil, fmt.Errorf("macrostore: read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, &s.cur); err != nil {
		return nil, fmt.Errorf("macrostore: parse %s: %w", path, err)
	}
	return s, nil
}

// Get returns the current target.
func (s *MacroStore) Get() DailyMacros {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cur
}

// Set replaces the target and writes it to disk atomically (temp file + rename).
func (s *MacroStore) Set(m DailyMacros) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("macrostore: encode: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".mydiet-*.json")
	if err != nil {
		return fmt.Errorf("macrostore: temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("macrostore: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("macrostore: close: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("macrostore: rename: %w", err)
	}

	s.cur = m
	return nil
}
