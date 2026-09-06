package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// fileShape mirrors the on-disk layout of data/ricette.json.
type fileShape struct {
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

// LoadData reads the recipe/ingredient JSON at path and returns them indexed
// by ID: recipes by Recipe.ID, ingredients by Ingredient.ID.
func LoadData(path string) (recipes map[string]Recipe, ingredients map[string]Ingredient, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("loaddata: read %s: %w", path, err)
	}

	var f fileShape
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
