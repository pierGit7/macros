// Package utils loads and models the recipe and ingredient data.
package utils

// Ingredient is a single food item and its macronutrient values,
// as provided by data/ricette.json (per that file's unit of measurement).
type Ingredient struct {
	ID       string  `json:"_id"`
	Name     string  `json:"name"`
	Calories float64 `json:"calories"`
	Protein  float64 `json:"protein"`
	Carbs    float64 `json:"carbs"`
	Fats     float64 `json:"fats"`
}

// Recipe is a meal composed of ingredients, referenced by their IDs.
// Look the IDs up in the ingredients map returned alongside it by LoadData.
type Recipe struct {
	ID            string   `json:"_id"`
	Name          string   `json:"name"`
	Steps         []string `json:"steps"`
	Meal          []string `json:"meal"`
	IngredientIDs []string `json:"ingredientIds"` // populated from ingredients[].ingredient
}
