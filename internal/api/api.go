// Package api serves the HTTP endpoints for the macros backend.
package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"macros/internal/utils"
)

// API serves the HTTP endpoints backed by the in-memory recipe/ingredient data.
type API struct {
	recipes            map[string]utils.Recipe
	ingredients        map[string]utils.Ingredient
	proteinIngridients map[string]utils.Ingredient
	carbIngridients    map[string]utils.Ingredient
	fatIngridients     map[string]utils.Ingredient
}

// New builds an API over the given data maps (as returned by utils.LoadData).
func New(recipes map[string]utils.Recipe, ingredients map[string]utils.Ingredient, proteinIngridients map[string]utils.Ingredient,
	carbIngridients map[string]utils.Ingredient, fatIngridients map[string]utils.Ingredient) *API {
	return &API{recipes: recipes, ingredients: ingredients, proteinIngridients: proteinIngridients, carbIngridients: carbIngridients, fatIngridients: fatIngridients}
}

// Router returns the configured chi router.
func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	//All endpoints
	r.Get("/recipes", a.handleListRecipes)
	r.Get("/ingredients/protein", a.handleProteinIngredients)
	r.Get("/ingredients/carb", a.handleCarbIngredients)
	r.Get("/ingredients/fat", a.handleFatIngredients)
	return r
}

// handleListRecipes responds with every recipe as a JSON object keyed by ID.
func (a *API) handleListRecipes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.recipes)
}

// handleProteinIngredients responds with the protein-source ingredients keyed by ID.
func (a *API) handleProteinIngredients(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.proteinIngridients)
}

// handleCarbIngredients responds with the carb-source ingredients keyed by ID.
func (a *API) handleCarbIngredients(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.carbIngridients)
}

// handleFatIngredients responds with the fat-source ingredients keyed by ID.
func (a *API) handleFatIngredients(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.fatIngridients)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api: encode response: %v", err) // headers already sent
	}
}
