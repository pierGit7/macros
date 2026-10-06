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

// DailyMacros is the user's body weight (kg, 0 = not set) and daily macronutrient
// target, in grams, together with the values derived from it: total kcal and the
// kcal of each meal. Build it with ComputeDailyMacros so the derived fields stay
// consistent.
type DailyMacros struct {
	Weight      float64 `json:"weight"`
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
func ComputeDailyMacros(weight, proteins, fats, carbs float64, mealsPerDay int) (DailyMacros, error) {
	if weight < 0 {
		return DailyMacros{}, errors.New("weight must not be negative")
	}
	if proteins < 0 || fats < 0 || carbs < 0 {
		return DailyMacros{}, errors.New("macros must not be negative")
	}
	if mealsPerDay < MinMealsPerDay || mealsPerDay > MaxMealsPerDay {
		return DailyMacros{}, fmt.Errorf("mealsPerDay must be between %d and %d", MinMealsPerDay, MaxMealsPerDay)
	}

	total := int(math.Round(KcalFromMacros(proteins, carbs, fats)))
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
		Weight:      weight,
		Proteins:    proteins,
		Fats:        fats,
		Carbs:       carbs,
		MealsPerDay: mealsPerDay,
		Kcal:        total,
		Meals:       meals,
	}, nil
}

// MealTarget returns the kcal and macros the named meal should provide: the daily
// grams scaled by the meal's share. ok is false if the plan has no such meal.
func (d DailyMacros) MealTarget(name string) (target Macros, ok bool) {
	for _, m := range d.Meals {
		if m.Name == name {
			f := float64(m.Pct) / 100
			return Macros{Kcal: float64(m.Kcal), Proteins: d.Proteins * f, Carbs: d.Carbs * f, Fats: d.Fats * f}, true
		}
	}
	return Macros{}, false
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
		s.cur, err = ComputeDailyMacros(0, 0, 0, 0, DefaultMealsPerDay)
		return s, err
	case err != nil:
		return nil, fmt.Errorf("macrostore: read %s: %w", path, err)
	}

	saved := DailyMacros{MealsPerDay: DefaultMealsPerDay}
	if err := json.Unmarshal(raw, &saved); err != nil {
		return nil, fmt.Errorf("macrostore: parse %s: %w", path, err)
	}
	s.cur, err = ComputeDailyMacros(saved.Weight, saved.Proteins, saved.Fats, saved.Carbs, saved.MealsPerDay)
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

	if err := writeFileAtomic(s.path, m); err != nil {
		return fmt.Errorf("macrostore: %w", err)
	}
	s.cur = m
	return nil
}

// writeFileAtomic writes v as indented JSON to path via a temp file in the same
// directory and a rename, so a crash never leaves a half-written file.
func writeFileAtomic(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
