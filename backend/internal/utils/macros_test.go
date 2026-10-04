package utils_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"macros/internal/utils"
)

func TestMacroStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mydiet.json")

	// missing file -> zero target, no file created
	store, err := utils.NewMacroStore(path)
	if err != nil {
		t.Fatalf("NewMacroStore (missing): %v", err)
	}
	zero, err := utils.ComputeDailyMacros(0, 0, 0, 0, utils.DefaultMealsPerDay)
	if err != nil {
		t.Fatalf("ComputeDailyMacros(zero): %v", err)
	}
	if got := store.Get(); !reflect.DeepEqual(got, zero) {
		t.Errorf("Get on missing file = %+v, want %+v", got, zero)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file created before Set: stat err = %v", err)
	}

	// Set persists to disk
	want, err := utils.ComputeDailyMacros(78, 140, 45, 220, 4) // non-zero weight: must survive the reopen too
	if err != nil {
		t.Fatalf("ComputeDailyMacros: %v", err)
	}
	if err := store.Set(want); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file missing after Set: %v", err)
	}

	// a fresh store (simulating a restart) reads back the saved value
	reopened, err := utils.NewMacroStore(path)
	if err != nil {
		t.Fatalf("NewMacroStore (reopen): %v", err)
	}
	if got := reopened.Get(); !reflect.DeepEqual(got, want) {
		t.Errorf("Get after reopen = %+v, want %+v", got, want)
	}
}

func TestComputeDailyMacros(t *testing.T) {
	// 150 P + 200 C + 50 F = 600 + 800 + 450 = 1850 kcal
	for _, tc := range []struct {
		meals int
		want  []utils.Meal
	}{
		{3, []utils.Meal{{"breakfast", 30, 555}, {"lunch", 40, 740}, {"dinner", 30, 555}}},
		{4, []utils.Meal{{"breakfast", 25, 463}, {"snack", 10, 185}, {"lunch", 35, 647}, {"dinner", 30, 555}}},
		{5, []utils.Meal{{"breakfast", 25, 463}, {"morningSnack", 10, 185}, {"lunch", 30, 554}, {"afternoonSnack", 10, 185}, {"dinner", 25, 463}}},
	} {
		t.Run(fmt.Sprintf("%d meals", tc.meals), func(t *testing.T) {
			got, err := utils.ComputeDailyMacros(80, 150, 50, 200, tc.meals)
			if err != nil {
				t.Fatalf("ComputeDailyMacros: %v", err)
			}
			if got.Kcal != 1850 {
				t.Errorf("Kcal = %d, want 1850", got.Kcal)
			}
			if got.MealsPerDay != tc.meals {
				t.Errorf("MealsPerDay = %d, want %d", got.MealsPerDay, tc.meals)
			}
			if !reflect.DeepEqual(got.Meals, tc.want) {
				t.Errorf("Meals = %+v, want %+v", got.Meals, tc.want)
			}
			sum := 0
			for _, m := range got.Meals {
				sum += m.Kcal
			}
			if sum != got.Kcal {
				t.Errorf("meals sum to %d, want total %d", sum, got.Kcal)
			}
		})
	}

	t.Run("zero macros", func(t *testing.T) {
		got, err := utils.ComputeDailyMacros(0, 0, 0, 0, 4)
		if err != nil || got.Kcal != 0 || len(got.Meals) != 4 {
			t.Errorf("got %+v, %v; want 0 kcal and 4 meals", got, err)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		for name, args := range map[string]struct {
			w, p, f, c float64
			meals      int
		}{
			"2 meals":         {80, 1, 1, 1, 2},
			"6 meals":         {80, 1, 1, 1, 6},
			"0 meals":         {80, 1, 1, 1, 0},
			"negative macro":  {80, -1, 1, 1, 4},
			"negative weight": {-1, 1, 1, 1, 4},
		} {
			if _, err := utils.ComputeDailyMacros(args.w, args.p, args.f, args.c, args.meals); err == nil {
				t.Errorf("%s: expected error, got nil", name)
			}
		}
	})
}

func TestMacroStoreUpgradesOldFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mydiet.json")
	// legacy file: macros only, no mealsPerDay / kcal / meals
	if err := os.WriteFile(path, []byte(`{"proteins":150,"fats":50,"carbs":200}`), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := utils.NewMacroStore(path)
	if err != nil {
		t.Fatalf("NewMacroStore: %v", err)
	}
	got := store.Get()
	if got.MealsPerDay != utils.DefaultMealsPerDay || got.Kcal != 1850 || len(got.Meals) != 4 {
		t.Errorf("upgraded = %+v, want default 4 meals and 1850 kcal", got)
	}
}
