// Package utils loads and models the recipe and ingredient data.
package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

// Recipe is a meal composed of ingredients, referenced by their IDs.
// Look the IDs up in the ingredients map returned alongside it by LoadDataRecipe.
type Recipe struct {
	ID            string   `json:"_id"`
	Name          string   `json:"name"`
	Steps         []string `json:"steps"`
	Meal          []string `json:"meal"`
	IngredientIDs []string `json:"ingredientIds"` // populated from ingredients[].ingredient
}

// RecipeSummary is the short form of a Recipe: just enough to list it and fetch it later.
type RecipeSummary struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
}

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

// RecipesForMeal returns the recipes suitable for meal, sorted by name.
// The meal-plan snack slots (morningSnack, afternoonSnack) match "snack" recipes.
func RecipesForMeal(recipes map[string]Recipe, meal string) []RecipeSummary {
	if meal == "morningSnack" || meal == "afternoonSnack" {
		meal = "snack"
	}
	out := []RecipeSummary{} // non-nil: encodes as [] not null
	for _, r := range recipes {
		if slices.Contains(r.Meal, meal) {
			out = append(out, RecipeSummary{ID: r.ID, Name: r.Name})
		}
	}
	// map iteration is random; keep output stable
	slices.SortFunc(out, func(a, b RecipeSummary) int { return strings.Compare(a.Name, b.Name) })
	return out
}
