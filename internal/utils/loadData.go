package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// fileShapeRecipe mirrors the on-disk layout of data/ricette.json.
type fileShapeRecipe struct {
	RecipesAvailable []struct {
		ID          string   `json:"_id"` //back tick is metadata to match json file
		Name        string   `json:"name"`
		Steps       []string `json:"steps"`
		Meal        []string `json:"meal"`
		Ingredients []struct {
			Ingredient string `json:"ingredient"`
		} `json:"ingredients"`
	} `json:"recipesAvailable"`
	IngredientsAvailable []Ingredient `json:"ingredientsAvailable"`
}

// fileShapeIngridients mirrors the on-disk layout of data/macro-sources.json.
type fileShapeIngridients struct {
	ProteinList []Ingredient `json:"proteinSources"`
	CarbList    []Ingredient `json:"carbSources"`
	FatList     []Ingredient `json:"fatSources"`
}

// LoadDataRecipe reads the recipe/ingredient JSON at path and returns them indexed
// by ID: recipes by Recipe.ID, ingredients by Ingredient.ID.
func LoadDataRecipe(path string) (recipes map[string]Recipe, ingredients map[string]Ingredient, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("loaddata: read %s: %w", path, err)
	}

	var f fileShapeRecipe
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, nil, fmt.Errorf("loaddata: parse %s: %w", path, err)
	}

	ingredients = make(map[string]Ingredient, len(f.IngredientsAvailable))
	for _, ing := range f.IngredientsAvailable {
		ingredients[ing.ID] = ing
	}

	recipes = make(map[string]Recipe, len(f.RecipesAvailable))
	for _, r := range f.RecipesAvailable {
		rec := Recipe{
			ID:            r.ID,
			Name:          r.Name,
			Steps:         r.Steps,
			Meal:          r.Meal,
			IngredientIDs: make([]string, 0, len(r.Ingredients)),
		}
		for _, ri := range r.Ingredients {
			rec.IngredientIDs = append(rec.IngredientIDs, ri.Ingredient)
		}
		recipes[rec.ID] = rec
	}

	return recipes, ingredients, nil
}

// LoadDataIngridients reads data/macro-sources.json at path and returns the
// protein, carb and fat source lists, each as a map keyed by Ingredient.ID.
func LoadDataIngridients(path string) (proteinIngridients map[string]Ingredient, carbIngridients map[string]Ingredient, fatIngridients map[string]Ingredient, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loaddata: read %s: %w", path, err)
	}

	var f fileShapeIngridients
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, nil, nil, fmt.Errorf("loaddata: parse %s: %w", path, err)
	}

	proteinIngridients = make(map[string]Ingredient, len(f.ProteinList))
	carbIngridients = make(map[string]Ingredient, len(f.CarbList))
	fatIngridients = make(map[string]Ingredient, len(f.FatList))
	for _, ing := range f.ProteinList {
		proteinIngridients[ing.ID] = ing
	}
	for _, ing := range f.CarbList {
		carbIngridients[ing.ID] = ing
	}
	for _, ing := range f.FatList {
		fatIngridients[ing.ID] = ing
	}

	return proteinIngridients, carbIngridients, fatIngridients, nil
}
