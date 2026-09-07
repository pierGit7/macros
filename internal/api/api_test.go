package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	srv := api.New(recipes, ingredients, proteinIngridients, carbIngridients, fatIngridients)

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
	srv := api.New(map[string]utils.Recipe{}, map[string]utils.Ingredient{}, map[string]utils.Ingredient{}, map[string]utils.Ingredient{}, map[string]utils.Ingredient{})

	req := httptest.NewRequest(http.MethodPost, "/recipes", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
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
	return api.New(recipes, ingredients, protein, carb, fat)
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
