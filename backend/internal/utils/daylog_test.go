package utils_test

import (
	"path/filepath"
	"testing"
	"time"

	"macros/internal/utils"
)

func TestLogStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.json")
	store, err := utils.NewLogStore(path)
	if err != nil {
		t.Fatalf("NewLogStore: %v", err)
	}
	day := time.Date(2026, 10, 4, 21, 0, 0, 0, time.Local)
	store.SetClock(func() time.Time { return day })

	if d := store.Today(); d.Date != "2026-10-04" || len(d.Meals) != 0 || len(d.Extras) != 0 {
		t.Fatalf("empty today = %+v", d)
	}

	lunch := utils.EatenMeal{RecipeID: "r1", RecipeName: "Pasta", Portion: utils.Portion{Macros: utils.Macros{Kcal: 700}}}
	if _, err := store.SetMeal("lunch", lunch); err != nil {
		t.Fatalf("SetMeal: %v", err)
	}
	d, err := store.AddExtra(utils.NewManualExtra("  Cappuccino ", 60))
	if err != nil {
		t.Fatalf("AddExtra: %v", err)
	}
	if len(d.Extras) != 1 || d.Extras[0].ID == "" || d.Extras[0].Name != "Cappuccino" {
		t.Fatalf("extras = %+v, want one trimmed extra with an ID", d.Extras)
	}
	extraID := d.Extras[0].ID

	// a fresh store (restart) reads the same day back
	reopened, err := utils.NewLogStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	reopened.SetClock(func() time.Time { return day })
	d = reopened.Today()
	if d.Meals["lunch"].RecipeName != "Pasta" || len(d.Extras) != 1 {
		t.Fatalf("after reopen = %+v", d)
	}

	// next day starts empty; the previous day is kept
	reopened.SetClock(func() time.Time { return day.Add(4 * time.Hour) }) // 01:00 on the 5th
	if d := reopened.Today(); d.Date != "2026-10-05" || len(d.Meals) != 0 || len(d.Extras) != 0 {
		t.Fatalf("next day = %+v, want empty", d)
	}
	reopened.SetClock(func() time.Time { return day })

	if _, found, err := reopened.RemoveExtra("nope"); err != nil || found {
		t.Errorf("RemoveExtra(unknown) = found %v, err %v", found, err)
	}
	if d, found, err := reopened.RemoveExtra(extraID); err != nil || !found || len(d.Extras) != 0 {
		t.Errorf("RemoveExtra = %+v, found %v, err %v", d.Extras, found, err)
	}
	if d, err := reopened.RemoveMeal("lunch"); err != nil || len(d.Meals) != 0 {
		t.Errorf("RemoveMeal = %+v, err %v", d.Meals, err)
	}
}

func TestBuildToday(t *testing.T) {
	plan, err := utils.ComputeDailyMacros(78, 156, 64, 240, 4) // 2160 kcal
	if err != nil {
		t.Fatal(err)
	}
	log := utils.DayLog{
		Date: "2026-10-04",
		Meals: map[string]utils.EatenMeal{
			"lunch":        {RecipeName: "Pasta", Portion: utils.Portion{Macros: utils.Macros{Kcal: 756}}},
			"morningSnack": {RecipeName: "Yogurt", Portion: utils.Portion{Macros: utils.Macros{Kcal: 200}}}, // from a 5-meal plan
		},
		Extras: []utils.Extra{utils.NewManualExtra("Cappuccino", 60)},
	}
	got := utils.BuildToday(plan, log)

	if got.TargetKcal != 2160 || got.EatenKcal != 1016 || got.LeftKcal != 1144 {
		t.Errorf("target/eaten/left = %d/%d/%d, want 2160/1016/1144", got.TargetKcal, got.EatenKcal, got.LeftKcal)
	}
	if len(got.Meals) != 5 || got.Meals[4].Name != "morningSnack" || got.Meals[4].Eaten == nil {
		t.Errorf("meals = %+v, want 4 planned + the stale morningSnack last", got.Meals)
	}
	if got.Meals[2].Name != "lunch" || got.Meals[2].Eaten == nil || got.Meals[0].Eaten != nil {
		t.Errorf("lunch should be eaten and breakfast not: %+v", got.Meals)
	}
}

func TestNewIngredientExtra(t *testing.T) {
	yogurt := utils.Ingredient{ID: "y", Name: "Yogurt greco 0%", NameEN: "Greek yogurt 0% fat", Protein: 10, Carbs: 4, Fats: 0.2}
	e := utils.NewIngredientExtra(yogurt, 150)
	// 150 g: 15 P, 6 C, 0.3 F -> 60 + 24 + 2.7 = 86.7 kcal
	if e.Kcal != 87 || e.Proteins != 15 || e.Name != "Greek yogurt 0% fat" || e.Grams != 150 || e.IngredientID != "y" {
		t.Errorf("extra = %+v", e)
	}
}
