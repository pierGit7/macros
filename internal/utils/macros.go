package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

// Allowed range and default for the number of meals the diet is split into.
const (
	MinMealsPerDay     = 3
	MaxMealsPerDay     = 5
	DefaultMealsPerDay = 4
)

// Meal is one eating occasion: its share of the daily kcal (Pct) and the
// resulting kcal. The presets below carry only Name and Pct; ComputeDailyMacros
// fills in Kcal.
type Meal struct {
	Name string `json:"name"`
	Pct  int    `json:"pct"`
	Kcal int    `json:"kcal"`
}

// DailyMacros is the user's daily macronutrient target, in grams, together with
// the values derived from it: total kcal and the kcal of each meal. Build it with
// ComputeDailyMacros so the derived fields stay consistent.
type DailyMacros struct {
	Proteins    float64 `json:"proteins"`
	Fats        float64 `json:"fats"`
	Carbs       float64 `json:"carbs"`
	MealsPerDay int     `json:"mealsPerDay"`
	Kcal        int     `json:"kcal"`
	Meals       []Meal  `json:"meals"`
}

// mealPresets gives, per meals-per-day, the meals in order and their kcal share.
// Each preset sums to 100 and has exactly one "lunch", which absorbs rounding.
var mealPresets = map[int][]Meal{
	3: {{Name: "breakfast", Pct: 30}, {Name: "lunch", Pct: 40}, {Name: "dinner", Pct: 30}},
	4: {{Name: "breakfast", Pct: 25}, {Name: "snack", Pct: 10}, {Name: "lunch", Pct: 35}, {Name: "dinner", Pct: 30}},
	5: {
		{Name: "breakfast", Pct: 25}, {Name: "morningSnack", Pct: 10}, {Name: "lunch", Pct: 30},
		{Name: "afternoonSnack", Pct: 10}, {Name: "dinner", Pct: 25},
	},
}

// ComputeDailyMacros validates the inputs and derives total kcal (4 kcal/g
// protein and carbs, 9 kcal/g fat) and the kcal of each meal. Per-meal values are
// rounded to whole kcal and lunch takes the remainder, so the meals always add up
// to the total.
func ComputeDailyMacros(proteins, fats, carbs float64, mealsPerDay int) (DailyMacros, error) {
	if proteins < 0 || fats < 0 || carbs < 0 {
		return DailyMacros{}, errors.New("macros must not be negative")
	}
	if mealsPerDay < MinMealsPerDay || mealsPerDay > MaxMealsPerDay {
		return DailyMacros{}, fmt.Errorf("mealsPerDay must be between %d and %d", MinMealsPerDay, MaxMealsPerDay)
	}

	total := int(math.Round(4*proteins + 4*carbs + 9*fats))
	meals := slices.Clone(mealPresets[mealsPerDay]) // copy: never mutate the preset table
	lunch, assigned := -1, 0
	for i := range meals {
		if meals[i].Name == "lunch" {
			lunch = i
			continue
		}
		meals[i].Kcal = int(math.Round(float64(total) * float64(meals[i].Pct) / 100))
		assigned += meals[i].Kcal
	}
	meals[lunch].Kcal = total - assigned

	return DailyMacros{
		Proteins:    proteins,
		Fats:        fats,
		Carbs:       carbs,
		MealsPerDay: mealsPerDay,
		Kcal:        total,
		Meals:       meals,
	}, nil
}

// MacroStore holds the daily macro target and persists it to a JSON file.
// It is safe for concurrent use.
type MacroStore struct {
	path string
	mu   sync.RWMutex
	cur  DailyMacros
}

// NewMacroStore loads the target from path. A missing file yields a zero target
// with the default meal split (and no file is created until the first Set); any
// other read or parse error is returned. Derived fields (kcal, meals) are always
// recomputed on load, which also upgrades files saved before they existed.
func NewMacroStore(path string) (*MacroStore, error) {
	s := &MacroStore{path: path}
	raw, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		s.cur, err = ComputeDailyMacros(0, 0, 0, DefaultMealsPerDay)
		return s, err
	case err != nil:
		return nil, fmt.Errorf("macrostore: read %s: %w", path, err)
	}

	saved := DailyMacros{MealsPerDay: DefaultMealsPerDay}
	if err := json.Unmarshal(raw, &saved); err != nil {
		return nil, fmt.Errorf("macrostore: parse %s: %w", path, err)
	}
	s.cur, err = ComputeDailyMacros(saved.Proteins, saved.Fats, saved.Carbs, saved.MealsPerDay)
	if err != nil {
		return nil, fmt.Errorf("macrostore: %s: %w", path, err)
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
