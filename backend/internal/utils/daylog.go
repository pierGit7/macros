package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// EatenMeal is a planned meal the user has eaten: the recipe they picked and the
// portion computed for it at that time.
type EatenMeal struct {
	RecipeID   string `json:"recipeId"`
	RecipeName string `json:"recipeName"`
	Portion
}

// Extra is food eaten outside the planned meals: either an ingredient the app knows
// and an amount (IngredientID set, macros computed), or a name and kcal typed in.
type Extra struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	IngredientID string  `json:"ingredientId,omitempty"`
	Grams        float64 `json:"grams,omitempty"`
	Macros
}

// DayLog is what was eaten on one day (Date is YYYY-MM-DD, local time).
type DayLog struct {
	Date   string               `json:"date"`
	Meals  map[string]EatenMeal `json:"meals"` // keyed by plan meal name
	Extras []Extra              `json:"extras"`
}

// LogStore keeps the day logs and persists them to a JSON file. It is safe for
// concurrent use.
type LogStore struct {
	path string
	mu   sync.Mutex
	days map[string]DayLog
	now  func() time.Time
}

// NewLogStore loads the logs from path; a missing file starts empty (and is created
// on the first change).
func NewLogStore(path string) (*LogStore, error) {
	s := &LogStore{path: path, days: map[string]DayLog{}, now: time.Now}
	raw, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return s, nil
	case err != nil:
		return nil, fmt.Errorf("logstore: read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, &s.days); err != nil {
		return nil, fmt.Errorf("logstore: parse %s: %w", path, err)
	}
	return s, nil
}

// SetClock replaces the clock that decides what "today" is (for tests).
func (s *LogStore) SetClock(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
}

// Today returns today's log (empty if nothing was eaten yet).
func (s *LogStore) Today() DayLog {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.today()
}

// SetMeal records the named meal as eaten, replacing an earlier choice.
func (s *LogStore) SetMeal(name string, m EatenMeal) (DayLog, error) {
	return s.update(func(d *DayLog) { d.Meals[name] = m })
}

// RemoveMeal marks the named meal as not eaten.
func (s *LogStore) RemoveMeal(name string) (DayLog, error) {
	return s.update(func(d *DayLog) { delete(d.Meals, name) })
}

// AddExtra appends e to today's extras, giving it a new ID.
func (s *LogStore) AddExtra(e Extra) (DayLog, error) {
	e.ID = newID()
	return s.update(func(d *DayLog) { d.Extras = append(d.Extras, e) })
}

// RemoveExtra deletes today's extra with the given ID; found is false if there is none.
func (s *LogStore) RemoveExtra(id string) (day DayLog, found bool, err error) {
	day, err = s.update(func(d *DayLog) {
		n := len(d.Extras)
		d.Extras = slices.DeleteFunc(d.Extras, func(e Extra) bool { return e.ID == id })
		found = len(d.Extras) < n
	})
	return day, found, err
}

// today returns a copy of today's log; callers hold s.mu.
func (s *LogStore) today() DayLog {
	date := s.now().Format(time.DateOnly)
	d := s.days[date]
	d.Date = date
	d.Meals = maps.Clone(d.Meals)
	if d.Meals == nil {
		d.Meals = map[string]EatenMeal{}
	}
	d.Extras = append([]Extra{}, d.Extras...)
	return d
}

// update applies change to today's log and persists all logs; on a write error the
// in-memory state is left unchanged.
func (s *LogStore) update(change func(*DayLog)) (DayLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	d := s.today()
	change(&d)
	days := maps.Clone(s.days)
	days[d.Date] = d
	if err := writeFileAtomic(s.path, days); err != nil {
		return DayLog{}, fmt.Errorf("logstore: %w", err)
	}
	s.days = days
	return s.today(), nil
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand.Read never fails on supported platforms
	return hex.EncodeToString(b)
}

// TodayMeal is one meal of today's plan and, if eaten, what was eaten.
type TodayMeal struct {
	Name     string     `json:"name"`
	Pct      int        `json:"pct"`
	PlanKcal int        `json:"planKcal"`
	Eaten    *EatenMeal `json:"eaten"`
}

// Today is the day at a glance: target, eaten and left kcal, every meal of the plan
// and the extras.
type Today struct {
	Date       string      `json:"date"`
	TargetKcal int         `json:"targetKcal"`
	EatenKcal  int         `json:"eatenKcal"`
	LeftKcal   int         `json:"leftKcal"` // negative when over the target
	Meals      []TodayMeal `json:"meals"`
	Extras     []Extra     `json:"extras"`
}

// BuildToday combines the plan with the day's log. Eaten meals that are no longer in
// the plan (meals per day changed after eating) still count and are listed last.
func BuildToday(plan DailyMacros, d DayLog) Today {
	t := Today{Date: d.Date, TargetKcal: plan.Kcal, Meals: []TodayMeal{}, Extras: d.Extras}
	var eaten float64
	inPlan := map[string]bool{}
	for _, m := range plan.Meals {
		inPlan[m.Name] = true
		tm := TodayMeal{Name: m.Name, Pct: m.Pct, PlanKcal: m.Kcal}
		if e, ok := d.Meals[m.Name]; ok {
			tm.Eaten = &e
			eaten += e.Kcal
		}
		t.Meals = append(t.Meals, tm)
	}
	var stale []string
	for name := range d.Meals {
		if !inPlan[name] {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	for _, name := range stale {
		e := d.Meals[name]
		t.Meals = append(t.Meals, TodayMeal{Name: name, Eaten: &e})
		eaten += e.Kcal
	}
	for _, e := range d.Extras {
		eaten += e.Kcal
	}
	t.EatenKcal = int(math.Round(eaten))
	t.LeftKcal = t.TargetKcal - t.EatenKcal
	return t
}

// NewIngredientExtra builds an extra for grams of ing, with macros computed.
func NewIngredientExtra(ing Ingredient, grams float64) Extra {
	m := ing.For(grams)
	m.Kcal = math.Round(m.Kcal)
	m.Proteins = math.Round(m.Proteins*10) / 10
	m.Carbs = math.Round(m.Carbs*10) / 10
	m.Fats = math.Round(m.Fats*10) / 10
	return Extra{Name: ing.NameEN, IngredientID: ing.ID, Grams: grams, Macros: m}
}

// NewManualExtra builds an extra typed in by hand: just a name and kcal.
func NewManualExtra(name string, kcal float64) Extra {
	return Extra{Name: strings.TrimSpace(name), Macros: Macros{Kcal: math.Round(kcal)}}
}
