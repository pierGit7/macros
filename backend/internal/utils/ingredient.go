package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

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

// fileShapeIngridients mirrors the on-disk layout of data/macro-sources.json.
type fileShapeIngridients struct {
	ProteinList []Ingredient `json:"proteinSources"`
	CarbList    []Ingredient `json:"carbSources"`
	FatList     []Ingredient `json:"fatSources"`
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
