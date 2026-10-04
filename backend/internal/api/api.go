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
	macros             *utils.MacroStore
}

// New builds an API over the given data maps (as returned by utils.LoadData) and
// the persistent daily-macro store.
func New(recipes map[string]utils.Recipe, ingredients map[string]utils.Ingredient, proteinIngridients map[string]utils.Ingredient,
	carbIngridients map[string]utils.Ingredient, fatIngridients map[string]utils.Ingredient, macros *utils.MacroStore) *API {
	return &API{
		recipes:            recipes,
		ingredients:        ingredients,
		proteinIngridients: proteinIngridients,
		carbIngridients:    carbIngridients,
		fatIngridients:     fatIngridients,
		macros:             macros,
	}
}

// Router returns the configured chi router.
func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	//All endpoints
	r.Get("/recipes", a.handleListRecipes)
	r.Get("/recipes/meal/{meal}", a.handleRecipesByMeal)
	r.Get("/ingredients/protein", a.handleProteinIngredients)
	r.Get("/ingredients/carb", a.handleCarbIngredients)
	r.Get("/ingredients/fat", a.handleFatIngredients)
	r.Get("/macros", a.handleGetMacros)
	r.Put("/macros", a.handlePutMacros)
	return r
}

// handleListRecipes responds with every recipe as a JSON object keyed by ID.
func (a *API) handleListRecipes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.recipes)
}

// handleRecipesByMeal responds with the id and name of every recipe suitable for
// the {meal} path parameter (e.g. breakfast), sorted by name; 404 if none match.
func (a *API) handleRecipesByMeal(w http.ResponseWriter, r *http.Request) {
	recipes := utils.RecipesForMeal(a.recipes, chi.URLParam(r, "meal"))
	if len(recipes) == 0 {
		writeError(w, http.StatusNotFound, "unknown meal")
		return
	}
	writeJSON(w, http.StatusOK, recipes)
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

// handleGetMacros responds with the user's current daily macro target.
func (a *API) handleGetMacros(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.macros.Get())
}

// handlePutMacros replaces and persists the body weight and daily macro target,
// deriving total kcal and per-meal kcal. weight and mealsPerDay are optional;
// mealsPerDay defaults to 4.
func (a *API) handlePutMacros(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Weight      float64 `json:"weight"`
		Proteins    float64 `json:"proteins"`
		Fats        float64 `json:"fats"`
		Carbs       float64 `json:"carbs"`
		MealsPerDay int     `json:"mealsPerDay"`
	}{MealsPerDay: utils.DefaultMealsPerDay}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	m, err := utils.ComputeDailyMacros(req.Weight, req.Proteins, req.Fats, req.Carbs, req.MealsPerDay)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.macros.Set(m); err != nil {
		log.Printf("api: save macros: %v", err)
		writeError(w, http.StatusInternalServerError, "could not save macros")
		return
	}
	writeJSON(w, http.StatusOK, a.macros.Get())
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api: encode response: %v", err) // headers already sent
	}
}
