package utils_test

import (
	"testing"

	"macros/internal/utils"
)

const dataPath = "../../data/ricette.json"

func TestLoadData(t *testing.T) {
	recipes, ingredients, err := utils.LoadData(dataPath)
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
	if _, _, err := utils.LoadData("does-not-exist.json"); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
