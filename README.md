# macros

Set your body weight and daily macros once; then each day pick a recipe per meal (the
app computes the grams to fit the meal), log extras, and see the kcal left.

```
backend/    Go API (recipes, ingredients, daily macro target)
frontend/   static pages (Today, Targets: HTML + CSS + JS) served by nginx
compose.yaml
```

## Run everything

```sh
docker compose up --build        # http://localhost:8080
```

A new day starts at midnight in `TZ` (default UTC), e.g. `TZ=Europe/Rome docker compose up -d`.

## Run the backend alone

```sh
cd backend
go run .        # http://localhost:8080
```

The recipe data (`data/ricette.json`, `data/macro-sources.json`) is private and not in
the repo: put your own copies in `backend/data/`, or point `RECIPES_PATH` and
`MACRO_SOURCES_PATH` at them. `backend/testdata/` has a small sample in the same format
(`RECIPES_PATH=testdata/ricette.json MACRO_SOURCES_PATH=testdata/macro-sources.json go run .`).
On the Pi the files live in `~/app/data/`, mounted read-only into the container.

Override the port with `PORT=9090 go run .`, the saved-targets file with
`DIET_PATH=/some/where/mydiet.json` (default `data/mydiet.json`) and the food log with
`LOG_PATH` (default `data/log.json`).

| Endpoint | Description |
|----------|-------------|
| `GET /recipes` | All recipes as a JSON object keyed by recipe ID |
| `GET /recipes/meal/{meal}` | `[{_id, name}]` of the recipes for a meal (`breakfast`, `lunch`, `dinner`, `snack`; `morningSnack`/`afternoonSnack` map to `snack`), sorted by name; 404 if none |
| `GET /ingredients` | Every ingredient (English name, unit, macros per 100 g), sorted by name |
| `GET /ingredients/protein` | Protein-source ingredients keyed by ID (also `/carb`, `/fat`) |
| `GET /macros` | Saved body weight and daily target, with total kcal and kcal per meal |
| `PUT /macros` | Replace it: `{"weight": 78, "proteins": 156, "carbs": 240, "fats": 64, "mealsPerDay": 4}` (`weight` optional, `mealsPerDay` 3-5, default 4) |
| `GET /today` | Today's target, eaten and left kcal, each planned meal (with what was eaten) and the extras |
| `GET /today/meals/{meal}/portion?recipe={id}` | Preview: the recipe scaled to that meal, grams per ingredient |
| `PUT /today/meals/{meal}` | Eat it: `{"recipeId": "..."}`; `DELETE` undoes |
| `POST /today/extras` | `{"ingredientId": "...", "grams": 150}` or by hand `{"name": "Cappuccino", "kcal": 60}` |
| `DELETE /today/extras/{id}` | Remove an extra |

kcal are always computed from macros (4/4/9 kcal per gram of protein/carbs/fats), so
portions and extras add up the same way as the daily target.

Through the frontend the same endpoints are under `/api`, e.g. `/api/macros`.

## Checks

```sh
cd backend && go vet ./... && go test ./... && golangci-lint run
```
