package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"macros/internal/api"
	"macros/internal/utils"
)

const dataPath = "../../data/ricette.json"

func TestHandleListRecipes(t *testing.T) {
	recipes, ingredients, err := utils.LoadData(dataPath)
	if err != nil {
		t.Fatalf("LoadData: %v", err)
	}
	srv := api.New(recipes, ingredients)

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
	srv := api.New(map[string]utils.Recipe{}, map[string]utils.Ingredient{})

	req := httptest.NewRequest(http.MethodPost, "/recipes", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
