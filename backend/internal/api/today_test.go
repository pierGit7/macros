package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"macros/internal/utils"
)

// call sends a request to the API and returns the status and raw body.
func call(t *testing.T, h http.Handler, method, path, body string) (int, []byte) {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, rd))
	return rec.Code, rec.Body.Bytes()
}

func decode[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return v
}

const chickenWithBread = "Petto di pollo con pane"

func recipeID(t *testing.T, h http.Handler, meal, name string) string {
	t.Helper()
	_, raw := call(t, h, http.MethodGet, "/recipes/meal/"+meal, "")
	for _, r := range decode[[]utils.RecipeSummary](t, raw) {
		if r.Name == name {
			return r.ID
		}
	}
	t.Fatalf("recipe %q not found for %s", name, meal)
	return ""
}

func TestTodayFlow(t *testing.T) {
	h := newTestAPI(t).Router()

	if code, _ := call(t, h, http.MethodPut, "/macros", `{"weight":78,"proteins":156,"fats":64,"carbs":240}`); code != http.StatusOK {
		t.Fatalf("PUT /macros = %d", code)
	}

	code, raw := call(t, h, http.MethodGet, "/today", "")
	today := decode[utils.Today](t, raw)
	if code != http.StatusOK || today.TargetKcal != 2160 || today.LeftKcal != 2160 || len(today.Meals) != 4 || today.Date == "" {
		t.Fatalf("GET /today = %d %+v", code, today)
	}

	id := recipeID(t, h, "lunch", chickenWithBread)

	// preview: nothing saved
	code, raw = call(t, h, http.MethodGet, "/today/meals/lunch/portion?recipe="+id, "")
	preview := decode[utils.EatenMeal](t, raw)
	if code != http.StatusOK || preview.RecipeName != chickenWithBread || preview.Kcal < 740 || preview.Kcal > 770 || len(preview.Items) != 4 {
		t.Fatalf("portion = %d %+v, want about 756 kcal and 4 items", code, preview)
	}
	if _, raw = call(t, h, http.MethodGet, "/today", ""); decode[utils.Today](t, raw).EatenKcal != 0 {
		t.Fatal("preview must not record anything")
	}

	// eat lunch
	code, raw = call(t, h, http.MethodPut, "/today/meals/lunch", `{"recipeId":"`+id+`"}`)
	today = decode[utils.Today](t, raw)
	if code != http.StatusOK || today.Meals[2].Eaten == nil || today.EatenKcal != int(preview.Kcal) || today.LeftKcal != 2160-int(preview.Kcal) {
		t.Fatalf("PUT lunch = %d %+v", code, today)
	}

	// extras: one from an ingredient, one by hand
	_, raw = call(t, h, http.MethodGet, "/ingredients", "")
	ings := decode[[]utils.Ingredient](t, raw)
	if len(ings) != 295 {
		t.Fatalf("GET /ingredients = %d items, want 295", len(ings))
	}
	var yogurt utils.Ingredient
	for _, ing := range ings {
		if ing.Name == "Yogurt greco 0%" {
			yogurt = ing
		}
	}
	code, raw = call(t, h, http.MethodPost, "/today/extras", `{"ingredientId":"`+yogurt.ID+`","grams":150}`)
	today = decode[utils.Today](t, raw)
	want := utils.NewIngredientExtra(yogurt, 150)
	if code != http.StatusOK || len(today.Extras) != 1 || today.Extras[0].Kcal != want.Kcal || today.Extras[0].Grams != 150 {
		t.Fatalf("POST ingredient extra = %d %+v, want %v kcal", code, today.Extras, want.Kcal)
	}
	code, raw = call(t, h, http.MethodPost, "/today/extras", `{"name":"Cappuccino","kcal":60}`)
	today = decode[utils.Today](t, raw)
	if code != http.StatusOK || len(today.Extras) != 2 || today.EatenKcal != int(preview.Kcal+want.Kcal+60) {
		t.Fatalf("POST manual extra = %d %+v", code, today)
	}

	// remove an extra, undo lunch
	code, raw = call(t, h, http.MethodDelete, "/today/extras/"+today.Extras[1].ID, "")
	if today = decode[utils.Today](t, raw); code != http.StatusOK || len(today.Extras) != 1 {
		t.Fatalf("DELETE extra = %d %+v", code, today.Extras)
	}
	code, raw = call(t, h, http.MethodDelete, "/today/meals/lunch", "")
	if today = decode[utils.Today](t, raw); code != http.StatusOK || today.Meals[2].Eaten != nil || today.EatenKcal != int(want.Kcal) {
		t.Fatalf("DELETE lunch = %d %+v", code, today)
	}
}

func TestTodayErrors(t *testing.T) {
	h := newTestAPI(t).Router()
	call(t, h, http.MethodPut, "/macros", `{"proteins":156,"fats":64,"carbs":240}`)
	id := recipeID(t, h, "lunch", chickenWithBread)

	for name, tc := range map[string]struct {
		method, path, body string
		want               int
	}{
		"portion unknown meal":   {http.MethodGet, "/today/meals/brunch/portion?recipe=" + id, "", http.StatusNotFound},
		"portion unknown recipe": {http.MethodGet, "/today/meals/lunch/portion?recipe=nope", "", http.StatusNotFound},
		"eat bad json":           {http.MethodPut, "/today/meals/lunch", `nope`, http.StatusBadRequest},
		"eat unknown recipe":     {http.MethodPut, "/today/meals/lunch", `{"recipeId":"nope"}`, http.StatusNotFound},
		"extra unknown":          {http.MethodPost, "/today/extras", `{"ingredientId":"nope","grams":10}`, http.StatusNotFound},
		"extra zero grams":       {http.MethodPost, "/today/extras", `{"ingredientId":"664879ef3f91ab24e2996d18","grams":0}`, http.StatusBadRequest},
		"extra no name":          {http.MethodPost, "/today/extras", `{"name":"  ","kcal":50}`, http.StatusBadRequest},
		"extra zero kcal":        {http.MethodPost, "/today/extras", `{"name":"Tea","kcal":0}`, http.StatusBadRequest},
		"remove unknown extra":   {http.MethodDelete, "/today/extras/nope", "", http.StatusNotFound},
		"extra bad json":         {http.MethodPost, "/today/extras", `{`, http.StatusBadRequest},
	} {
		if code, body := call(t, h, tc.method, tc.path, tc.body); code != tc.want {
			t.Errorf("%s: %d %s, want %d", name, code, body, tc.want)
		}
	}
}
