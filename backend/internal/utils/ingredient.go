package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

// Ingredient is a single food item and its macronutrient values per 100 g,
// as provided by data/ricette.json.
type Ingredient struct {
	ID       string  `json:"_id"`
	Name     string  `json:"name"`
	NameEN   string  `json:"nameEn"` // English name; falls back to Name
	Calories float64 `json:"calories"`
	Protein  float64 `json:"protein"`
	Carbs    float64 `json:"carbs"`
	Fats     float64 `json:"fats"`
	Unit     Unit    `json:"unit"`
}

// Unit is how an ingredient is usually counted (grams, pieces, spoons) and how many
// grams one of those is. It is for display only: recipe quantities are always grams.
type Unit struct {
	Type         string  `json:"type"`
	ValueToGrams float64 `json:"valueToGrams"`
}

// UnmarshalJSON reads the on-disk layout (nameI18n.enUs, unityOfMeasurement) into
// the flat NameEN and Unit fields.
func (i *Ingredient) UnmarshalJSON(raw []byte) error {
	type plain Ingredient // same fields, no UnmarshalJSON: avoids recursion
	var f struct {
		plain
		NameI18n struct {
			EnUs string `json:"enUs"`
		} `json:"nameI18n"`
		UnityOfMeasurement *Unit `json:"unityOfMeasurement"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	*i = Ingredient(f.plain)
	if i.NameEN == "" {
		i.NameEN = strings.TrimSpace(f.NameI18n.EnUs)
	}
	if i.NameEN == "" {
		i.NameEN = i.Name
	}
	if f.UnityOfMeasurement != nil {
		i.Unit = *f.UnityOfMeasurement
	}
	if i.Unit.Type == "" {
		i.Unit = Unit{Type: "grams", ValueToGrams: 1}
	}
	return nil
}

// Macros is an amount of food expressed as kcal and macronutrient grams.
type Macros struct {
	Kcal     float64 `json:"kcal"`
	Proteins float64 `json:"proteins"`
	Carbs    float64 `json:"carbs"`
	Fats     float64 `json:"fats"`
}

// KcalFromMacros is the energy of the given grams of protein, carbs and fats
// (4, 4 and 9 kcal per gram). The daily target is defined this way, so everything
// counted against it (portions, extras) uses it too.
func KcalFromMacros(proteins, carbs, fats float64) float64 {
	return 4*proteins + 4*carbs + 9*fats
}

// For returns the macros of grams of the ingredient. Kcal is derived from the
// macros, not taken from Calories: about 10% of the data's labelled values
// disagree with their own macros (e.g. one says 200 kcal for 604 kcal of macros).
func (i Ingredient) For(grams float64) Macros {
	f := grams / 100
	m := Macros{Proteins: i.Protein * f, Carbs: i.Carbs * f, Fats: i.Fats * f}
	m.Kcal = KcalFromMacros(m.Proteins, m.Carbs, m.Fats)
	return m
}

// IngredientList returns every ingredient sorted by English name, for searching.
func IngredientList(ingredients map[string]Ingredient) []Ingredient {
	out := make([]Ingredient, 0, len(ingredients))
	for _, ing := range ingredients {
		out = append(out, ing)
	}
	key := func(i Ingredient) string { return strings.ToLower(strings.TrimLeft(i.NameEN, `"' `)) } // `"Exquisia"` sorts under E
	slices.SortFunc(out, func(a, b Ingredient) int { return strings.Compare(key(a), key(b)) })
	return out
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
