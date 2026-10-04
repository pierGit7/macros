# macros

Set your body weight and daily macros, and see the calories split across your meals.

```
backend/    Go API (recipes, ingredients, daily macro target)
frontend/   static landing page (HTML + CSS + JS) served by nginx
compose.yaml
```

## Run everything

```sh
docker compose up --build        # http://localhost:8080
```

## Run the backend alone

```sh
cd backend
go run .        # http://localhost:8080
```

Override the port with `PORT=9090 go run .` and the saved-targets file with
`DIET_PATH=/some/where/mydiet.json` (default `data/mydiet.json`).

| Endpoint | Description |
|----------|-------------|
| `GET /recipes` | All recipes as a JSON object keyed by recipe ID |
| `GET /recipes/meal/{meal}` | `[{_id, name}]` of the recipes for a meal (`breakfast`, `lunch`, `dinner`, `snack`; `morningSnack`/`afternoonSnack` map to `snack`), sorted by name; 404 if none |
| `GET /ingredients/protein` | Protein-source ingredients keyed by ID (also `/carb`, `/fat`) |
| `GET /macros` | Saved body weight and daily target, with total kcal and kcal per meal |
| `PUT /macros` | Replace it: `{"weight": 78, "proteins": 156, "carbs": 240, "fats": 64, "mealsPerDay": 4}` (`weight` optional, `mealsPerDay` 3-5, default 4) |

Through the frontend the same endpoints are under `/api`, e.g. `/api/macros`.

## Checks

```sh
cd backend && go vet ./... && go test ./... && golangci-lint run
```
