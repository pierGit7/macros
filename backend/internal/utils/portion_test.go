package utils_test

import (
	"math"
	"testing"

	"macros/internal/utils"
)

func TestComputePortionKnownRecipe(t *testing.T) {
	recipes, ingredients, err := utils.LoadDataRecipe(dataPath)
	if err != nil {
		t.Fatalf("LoadDataRecipe: %v", err)
	}
	day, err := utils.ComputeDailyMacros(78, 156, 64, 240, 4)
	if err != nil {
		t.Fatal(err)
	}
	target, ok := day.MealTarget("lunch") // 35%
	if !ok {
		t.Fatal("no lunch in 4-meal plan")
	}

	var chicken utils.Recipe
	for _, r := range recipes {
		if r.Name == "Petto di pollo con pane" {
			chicken = r
		}
	}
	p, err := utils.ComputePortion(chicken, ingredients, target)
	if err != nil {
		t.Fatalf("ComputePortion: %v", err)
	}

	if math.Abs(p.Kcal-target.Kcal)/target.Kcal > 0.01 {
		t.Errorf("kcal = %v, want within 1%% of %v", p.Kcal, target.Kcal)
	}
	grams := map[string]float64{}
	for _, it := range p.Items {
		grams[it.Name] = it.Grams
	}
	// adjustable amounts computed by the solver; zucchini is fixed by the recipe
	for name, want := range map[string]float64{"Petto di pollo": 200, "Pane integrale": 175, "Olio d’oliva evo": 15, "Zucchine": 200} {
		if math.Abs(grams[name]-want) > 5 {
			t.Errorf("%s = %v g, want about %v g", name, grams[name], want)
		}
	}
	if p.Items[len(p.Items)-1].Fixed != true || p.Items[0].Fixed {
		t.Errorf("want adjustable items first, fixed last: %+v", p.Items)
	}
}

// Every recipe, in every meal it is meant for, should come within 15% of that
// meal's kcal, with no negative amounts. Not 0%: a recipe's fixed ingredients can
// already exceed a small meal (60 g of prosciutto has more protein than a 10% snack).
func TestComputePortionAllRecipes(t *testing.T) {
	recipes, ingredients, err := utils.LoadDataRecipe(dataPath)
	if err != nil {
		t.Fatalf("LoadDataRecipe: %v", err)
	}
	day, err := utils.ComputeDailyMacros(78, 156, 64, 240, 4)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, r := range recipes {
		for _, meal := range r.Meal {
			target, ok := day.MealTarget(meal)
			if !ok {
				t.Fatalf("recipe %s: meal %q not in the 4-meal plan", r.Name, meal)
			}
			p, err := utils.ComputePortion(r, ingredients, target)
			if err != nil {
				t.Fatalf("%s: %v", r.Name, err)
			}
			if off := math.Abs(p.Kcal-target.Kcal) / target.Kcal; off > 0.15 {
				t.Errorf("%s (%s): %v kcal, target %v (%.0f%% off)", r.Name, meal, p.Kcal, target.Kcal, off*100)
			}
			for _, it := range p.Items {
				if it.Grams < 0 {
					t.Errorf("%s: %s has %v g", r.Name, it.Name, it.Grams)
				}
			}
			checked++
		}
	}
	if checked != 300 {
		t.Errorf("checked %d recipe/meal pairs, want 300", checked)
	}
}

func TestComputePortionFixedOnly(t *testing.T) {
	g := 50.0
	ings := map[string]utils.Ingredient{"oats": {ID: "oats", Name: "Oats", Calories: 380, Protein: 13, Carbs: 60, Fats: 7}}
	r := utils.Recipe{ID: "r", Items: []utils.RecipeItem{{IngredientID: "oats", Grams: &g}}}
	p, err := utils.ComputePortion(r, ings, utils.Macros{Kcal: 600, Proteins: 30, Carbs: 80, Fats: 20})
	if err != nil {
		t.Fatal(err)
	}
	// 50 g: 6.5 P, 30 C, 3.5 F -> 26 + 120 + 31.5 = 177.5 kcal, rounded
	if len(p.Items) != 1 || p.Items[0].Grams != 50 || p.Kcal != 178 {
		t.Errorf("fixed-only recipe = %+v, want 50 g and 178 kcal unchanged", p)
	}
}

func TestComputePortionUnknownIngredient(t *testing.T) {
	r := utils.Recipe{ID: "r", Items: []utils.RecipeItem{{IngredientID: "missing"}}}
	if _, err := utils.ComputePortion(r, map[string]utils.Ingredient{}, utils.Macros{}); err == nil {
		t.Fatal("expected error for unknown ingredient")
	}
}
