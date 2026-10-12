// Command macros runs the HTTP server for the recipe/ingredient backend.
package main

import (
	"log"
	"net/http"
	"os"

	_ "time/tzdata" // zone database for TZ: the distroless image has none

	"macros/internal/api"
	"macros/internal/utils"
)

func main() {
	// recipe and ingredient data is private and not in the repo; RECIPES_PATH and
	// MACRO_SOURCES_PATH move it, e.g. onto a read-only mount (testdata/ has samples)
	recipesPath := "data/ricette.json"
	if p := os.Getenv("RECIPES_PATH"); p != "" {
		recipesPath = p
	}
	macroSourcesPath := "data/macro-sources.json"
	if p := os.Getenv("MACRO_SOURCES_PATH"); p != "" {
		macroSourcesPath = p
	}
	recipes, ingredients, err := utils.LoadDataRecipe(recipesPath) //load data from json file
	if err != nil {
		log.Fatal(err)
	}
	proteinIngridients, carbIngridients, fatIngridients, err := utils.LoadDataIngridients(macroSourcesPath)
	if err != nil {
		log.Fatal(err)
	}
	dietPath := "data/mydiet.json" // mutable user state; DIET_PATH moves it, e.g. onto a container volume
	if p := os.Getenv("DIET_PATH"); p != "" {
		dietPath = p
	}
	macroStore, err := utils.NewMacroStore(dietPath)
	if err != nil {
		log.Fatal(err)
	}
	logPath := "data/log.json" // what was eaten each day; LOG_PATH moves it like DIET_PATH
	if p := os.Getenv("LOG_PATH"); p != "" {
		logPath = p
	}
	logStore, err := utils.NewLogStore(logPath)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("loaded %d recipes, %d ingredients", len(recipes), len(ingredients))

	//start server logic
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	srv := api.New(recipes, ingredients, proteinIngridients, carbIngridients, fatIngridients, macroStore, logStore)
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Router()); err != nil {
		log.Fatal(err)
	}
}
