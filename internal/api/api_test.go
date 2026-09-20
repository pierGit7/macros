package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	get := func() (int, utils.DailyMacros) {
		req := httptest.NewRequest(http.MethodGet, "/macros", nil)
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, req)
		var m utils.DailyMacros
		if rec.Code == http.StatusOK {
			if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
				t.Fatalf("decode GET body: %v", err)
			}
		}
		return rec.Code, m
	}

	put := func(body string) int {
		req := httptest.NewRequest(http.MethodPut, "/macros", strings.NewReader(body))
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, req)
		return rec.Code
	}

	// first run: zero target
	if code, m := get(); code != http.StatusOK || m != (utils.DailyMacros{}) {
		t.Fatalf("initial GET = %d %+v, want 200 zero value", code, m)
	}

	// PUT replaces and persists
	want := utils.DailyMacros{Proteins: 150, Fats: 50, Carbs: 200}
	if code := put(`{"proteins":150,"fats":50,"carbs":200}`); code != http.StatusOK {
		t.Fatalf("PUT = %d, want 200", code)
	}
	if code, m := get(); code != http.StatusOK || m != want {
		t.Fatalf("GET after PUT = %d %+v, want 200 %+v", code, m, want)
	}

	// bad input
	if code := put(`not json`); code != http.StatusBadRequest {
		t.Errorf("PUT invalid body = %d, want 400", code)
	}
	if code := put(`{"proteins":-1,"fats":0,"carbs":0}`); code != http.StatusBadRequest {
		t.Errorf("PUT negative = %d, want 400", code)
	}
}
