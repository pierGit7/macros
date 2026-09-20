package utils_test

import (
	"os"
	"path/filepath"
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
	if got := store.Get(); got != (utils.DailyMacros{}) {
		t.Errorf("Get on missing file = %+v, want zero value", got)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file created before Set: stat err = %v", err)
	}

	// Set persists to disk
	want := utils.DailyMacros{Proteins: 140, Fats: 45, Carbs: 220}
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
	if got := reopened.Get(); got != want {
		t.Errorf("Get after reopen = %+v, want %+v", got, want)
	}
}
