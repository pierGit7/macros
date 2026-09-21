package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"macros/internal/api"
	"macros/internal/utils"
)

const recipePath = "../../data/ricette.json"
const ingrPath = "../../data/macro-sources.json"

func TestHandleListRecipes(t *testing.T) {
	recipes, ingredients, err := utils.LoadDataRecipe(recipePath)
	if err != nil {
		t.Fatalf("LoadDataRecipe: %v", err)
	}
	proteinIngridients, carbIngridients, fatIngridients, err := utils.LoadDataIngridients(ingrPath)
	if err != nil {
		t.Fatalf("LoadDataIngridients: %v", err)
	}
	srv := api.New(recipes, ingredients, proteinIngridients, carbIngridients, fatIngridients, testMacroStore(t))

	req := httptest.NewRequest(http.MethodGet, "/recipes", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got map[string]utils.Recipe
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != 138 {
		t.Errorf("recipe count = %d, want 138", len(got))
	}

	const knownID = "664cace39506e16394ef175a"
	r, ok := got[knownID]
	if !ok {
		t.Fatalf("recipe %s missing from response", knownID)
	}
	if len(r.IngredientIDs) != 5 {
		t.Errorf("recipe %s IngredientIDs len = %d, want 5", knownID, len(r.IngredientIDs))
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv := api.New(map[string]utils.Recipe{}, map[string]utils.Ingredient{}, map[string]utils.Ingredient{}, map[string]utils.Ingredient{}, map[string]utils.Ingredient{}, testMacroStore(t))

	req := httptest.NewRequest(http.MethodPost, "/recipes", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// testMacroStore returns a MacroStore backed by a fresh temp file.
func testMacroStore(t *testing.T) *utils.MacroStore {
	t.Helper()
	store, err := utils.NewMacroStore(t.TempDir() + "/mydiet.json")
	if err != nil {
		t.Fatalf("NewMacroStore: %v", err)
	}
	return store
}

// newTestAPI builds an API backed by the real data files.
func newTestAPI(t *testing.T) *api.API {
	t.Helper()
	recipes, ingredients, err := utils.LoadDataRecipe(recipePath)
	if err != nil {
		t.Fatalf("LoadDataRecipe: %v", err)
	}
	protein, carb, fat, err := utils.LoadDataIngridients(ingrPath)
	if err != nil {
		t.Fatalf("LoadDataIngridients: %v", err)
	}
	return api.New(recipes, ingredients, protein, carb, fat, testMacroStore(t))
}

func TestHandleIngredientEndpoints(t *testing.T) {
	srv := newTestAPI(t)

	for _, tc := range []struct {
		path string
		want int
	}{
		{"/ingredients/protein", 99},
		{"/ingredients/carb", 152},
		{"/ingredients/fat", 44},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			srv.Router().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var got map[string]utils.Ingredient
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if len(got) != tc.want {
				t.Errorf("%s count = %d, want %d", tc.path, len(got), tc.want)
			}
			for id, ing := range got {
				if ing.ID != id {
					t.Errorf("%s: entry keyed %q has ID %q", tc.path, id, ing.ID)
				}
			}
		})
	}
}

func TestIngredientEndpointMethodNotAllowed(t *testing.T) {
	srv := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/ingredients/protein", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestMacrosEndpoints(t *testing.T) {
	srv := newTestAPI(t)

	do := func(method, body string) (int, utils.DailyMacros) {
		var rd io.Reader
		if body != "" {
			rd = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, "/macros", rd)
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, req)
		var m utils.DailyMacros
		if rec.Code == http.StatusOK {
			if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
				t.Fatalf("decode %s body: %v", method, err)
			}
		}
		return rec.Code, m
	}
	names := func(m utils.DailyMacros) []string {
		out := make([]string, len(m.Meals))
		for i, meal := range m.Meals {
			out[i] = meal.Name
		}
		return out
	}
	kcals := func(m utils.DailyMacros) []int {
		out := make([]int, len(m.Meals))
		for i, meal := range m.Meals {
			out[i] = meal.Kcal
		}
		return out
	}

	// first run: zero target with the default 4-meal structure
	code, m := do(http.MethodGet, "")
	if code != http.StatusOK || m.Kcal != 0 || m.MealsPerDay != 4 || len(m.Meals) != 4 {
		t.Fatalf("initial GET = %d %+v, want 200, 0 kcal, 4 meals", code, m)
	}

	// PUT without mealsPerDay -> default 4, kcal + meals derived and returned
	code, m = do(http.MethodPut, `{"proteins":150,"fats":50,"carbs":200}`)
	if code != http.StatusOK {
		t.Fatalf("PUT = %d, want 200", code)
	}
	if m.Kcal != 1850 || m.MealsPerDay != 4 || !reflect.DeepEqual(kcals(m), []int{463, 185, 647, 555}) {
		t.Fatalf("PUT body = %+v, want 1850 kcal, 4 meals 463/185/647/555", m)
	}
	if want := []string{"breakfast", "snack", "lunch", "dinner"}; !reflect.DeepEqual(names(m), want) {
		t.Errorf("meal names = %v, want %v", names(m), want)
	}

	// GET returns the same persisted body
	if code, got := do(http.MethodGet, ""); code != http.StatusOK || !reflect.DeepEqual(got, m) {
		t.Fatalf("GET after PUT = %d %+v, want 200 %+v", code, got, m)
	}

	// explicit meal counts
	code, m = do(http.MethodPut, `{"proteins":150,"fats":50,"carbs":200,"mealsPerDay":3}`)
	if code != http.StatusOK || !reflect.DeepEqual(kcals(m), []int{555, 740, 555}) {
		t.Errorf("3 meals: %d %+v", code, m)
	}
	code, m = do(http.MethodPut, `{"proteins":150,"fats":50,"carbs":200,"mealsPerDay":5}`)
	if code != http.StatusOK || !reflect.DeepEqual(kcals(m), []int{463, 185, 554, 185, 463}) {
		t.Errorf("5 meals: %d %+v", code, m)
	}

	// bad input
	for name, body := range map[string]string{
		"invalid json":   `not json`,
		"negative macro": `{"proteins":-1,"fats":0,"carbs":0}`,
		"2 meals":        `{"proteins":1,"fats":1,"carbs":1,"mealsPerDay":2}`,
		"6 meals":        `{"proteins":1,"fats":1,"carbs":1,"mealsPerDay":6}`,
		"0 meals":        `{"proteins":1,"fats":1,"carbs":1,"mealsPerDay":0}`,
	} {
		if code, _ := do(http.MethodPut, body); code != http.StatusBadRequest {
			t.Errorf("PUT %s = %d, want 400", name, code)
		}
	}
}
