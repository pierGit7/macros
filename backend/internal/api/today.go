package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"macros/internal/utils"
)

// handleToday responds with today's target, eaten and left kcal, the planned meals
// (with what was eaten) and the extras.
func (a *API) handleToday(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, utils.BuildToday(a.macros.Get(), a.logs.Today()))
}

// handlePortion responds with the recipe given by ?recipe= scaled to the {meal} of
// the plan, without saving anything: a preview for the recipe chooser.
func (a *API) handlePortion(w http.ResponseWriter, r *http.Request) {
	m, status, err := a.portionFor(chi.URLParam(r, "meal"), r.URL.Query().Get("recipe"))
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// handleEatMeal records {meal} as eaten with the recipe in the body, scaled to the
// meal, and responds with the updated day.
func (a *API) handleEatMeal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecipeID string `json:"recipeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	meal := chi.URLParam(r, "meal")
	m, status, err := a.portionFor(meal, req.RecipeID)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	a.saveDay(w, func() (utils.DayLog, error) { return a.logs.SetMeal(meal, m) })
}

// handleUneatMeal marks {meal} as not eaten (undo) and responds with the updated day.
func (a *API) handleUneatMeal(w http.ResponseWriter, r *http.Request) {
	meal := chi.URLParam(r, "meal")
	a.saveDay(w, func() (utils.DayLog, error) { return a.logs.RemoveMeal(meal) })
}

// handleAddExtra adds food eaten outside the meals: {ingredientId, grams} for an
// ingredient the app knows, or {name, kcal} typed in by hand.
func (a *API) handleAddExtra(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IngredientID string  `json:"ingredientId"`
		Grams        float64 `json:"grams"`
		Name         string  `json:"name"`
		Kcal         float64 `json:"kcal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var e utils.Extra
	if req.IngredientID != "" {
		ing, ok := a.ingredients[req.IngredientID]
		if !ok {
			writeError(w, http.StatusNotFound, "unknown ingredient")
			return
		}
		if req.Grams <= 0 {
			writeError(w, http.StatusBadRequest, "grams must be more than 0")
			return
		}
		e = utils.NewIngredientExtra(ing, req.Grams)
	} else {
		e = utils.NewManualExtra(req.Name, req.Kcal)
		if e.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
		if req.Kcal <= 0 {
			writeError(w, http.StatusBadRequest, "kcal must be more than 0")
			return
		}
	}
	a.saveDay(w, func() (utils.DayLog, error) { return a.logs.AddExtra(e) })
}

// handleRemoveExtra deletes today's extra {id} and responds with the updated day.
func (a *API) handleRemoveExtra(w http.ResponseWriter, r *http.Request) {
	day, found, err := a.logs.RemoveExtra(chi.URLParam(r, "id"))
	if err == nil && !found {
		writeError(w, http.StatusNotFound, "unknown extra")
		return
	}
	a.saveDay(w, func() (utils.DayLog, error) { return day, err })
}

// portionFor scales the recipe to the plan's meal; on failure it also returns the
// HTTP status to answer with.
func (a *API) portionFor(meal, recipeID string) (utils.EatenMeal, int, error) {
	target, ok := a.macros.Get().MealTarget(meal)
	if !ok {
		return utils.EatenMeal{}, http.StatusNotFound, errors.New("unknown meal")
	}
	recipe, ok := a.recipes[recipeID]
	if !ok {
		return utils.EatenMeal{}, http.StatusNotFound, errors.New("unknown recipe")
	}
	p, err := utils.ComputePortion(recipe, a.ingredients, target)
	if err != nil {
		log.Printf("api: %v", err)
		return utils.EatenMeal{}, http.StatusInternalServerError, errors.New("could not compute portion")
	}
	return utils.EatenMeal{RecipeID: recipe.ID, RecipeName: recipe.Name, Portion: p}, http.StatusOK, nil
}

// saveDay runs a change to today's log and responds with the resulting day view.
func (a *API) saveDay(w http.ResponseWriter, change func() (utils.DayLog, error)) {
	day, err := change()
	if err != nil {
		log.Printf("api: save day: %v", err)
		writeError(w, http.StatusInternalServerError, "could not save")
		return
	}
	writeJSON(w, http.StatusOK, utils.BuildToday(a.macros.Get(), day))
}
