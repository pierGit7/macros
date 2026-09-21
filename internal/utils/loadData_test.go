package utils_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"macros/internal/utils"
)

const dataPath = "../../data/ricette.json"
const macroSourcesPath = "../../data/macro-sources.json"

func TestLoadData(t *testing.T) {
	recipes, ingredients, err := utils.LoadDataRecipe(dataPath)

	if err != nil {
		t.Fatalf("LoadData: %v", err)
	}

	if got, want := len(recipes), 138; got != want {
		t.Errorf("recipe count = %d, want %d", got, want)
	}
	if got, want := len(ingredients), 295; got != want {
		t.Errorf("ingredient count = %d, want %d", got, want)
	}

	const knownID = "664cace39506e16394ef175a"
	r, ok := recipes[knownID]
	if !ok {
		t.Fatalf("recipe %s not found in map", knownID)
	}
	if r.ID != knownID {
		t.Errorf("recipe.ID = %q, want %q", r.ID, knownID)
	}
	if r.Name != "Filetti di pesce impanati" {
		t.Errorf("recipe.Name = %q, want %q", r.Name, "Filetti di pesce impanati")
	}
	if got, want := len(r.IngredientIDs), 5; got != want {
		t.Fatalf("recipe.IngredientIDs len = %d, want %d", got, want)
	}
	for _, id := range r.IngredientIDs {
		if _, ok := ingredients[id]; !ok {
			t.Errorf("ingredient %s referenced by recipe %s missing from ingredients map", id, knownID)
		}
	}
}

func TestLoadDataMissingFile(t *testing.T) {
	if _, _, err := utils.LoadDataRecipe("does-not-exist.json"); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadDataIngridients(t *testing.T) {
	protein, carb, fat, err := utils.LoadDataIngridients(macroSourcesPath)
	if err != nil {
		t.Fatalf("LoadDataIngridients: %v", err)
	}

	for _, tc := range []struct {
		name string
		got  int
		want int
	}{
		{"protein", len(protein), 99},
		{"carb", len(carb), 152},
		{"fat", len(fat), 44},
	} {
		if tc.got != tc.want {
			t.Errorf("%s source count = %d, want %d", tc.name, tc.got, tc.want)
		}
	}

	// every ingredient sits in exactly one list
	seen := make(map[string]int)
	for _, m := range []map[string]utils.Ingredient{protein, carb, fat} {
		for id := range m {
			seen[id]++
		}
	}
	if len(seen) != 295 {
		t.Errorf("distinct ingredients across lists = %d, want 295", len(seen))
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("ingredient %s appears in %d lists, want 1", id, n)
		}
	}

	// a known lean protein source is keyed by its ID and carries its macros
	const tunaID = "664879ef3f91ab24e2996d18"
	tuna, ok := protein[tunaID]
	if !ok {
		t.Fatalf("protein map missing %s (Trancio di tonno)", tunaID)
	}
	if tuna.Name != "Trancio di tonno" {
		t.Errorf("protein[%s].Name = %q, want %q", tunaID, tuna.Name, "Trancio di tonno")
	}
	if tuna.Protein <= 0 {
		t.Errorf("protein[%s].Protein = %v, want > 0", tunaID, tuna.Protein)
	}
}

func TestLoadDataIngridientsMissingFile(t *testing.T) {
	if _, _, _, err := utils.LoadDataIngridients("does-not-exist.json"); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestMacroStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mydiet.json")

	// missing file -> zero target, no file created
	store, err := utils.NewMacroStore(path)
	if err != nil {
		t.Fatalf("NewMacroStore (missing): %v", err)
	}
	zero, err := utils.ComputeDailyMacros(0, 0, 0, utils.DefaultMealsPerDay)
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
	want, err := utils.ComputeDailyMacros(140, 45, 220, 4)
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
			got, err := utils.ComputeDailyMacros(150, 50, 200, tc.meals)
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
		got, err := utils.ComputeDailyMacros(0, 0, 0, 4)
		if err != nil || got.Kcal != 0 || len(got.Meals) != 4 {
			t.Errorf("got %+v, %v; want 0 kcal and 4 meals", got, err)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		for name, args := range map[string]struct {
			p, f, c float64
			meals   int
		}{
			"2 meals":        {1, 1, 1, 2},
			"6 meals":        {1, 1, 1, 6},
			"0 meals":        {1, 1, 1, 0},
			"negative macro": {-1, 1, 1, 4},
		} {
			if _, err := utils.ComputeDailyMacros(args.p, args.f, args.c, args.meals); err == nil {
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
