// Command macros runs the HTTP server for the recipe/ingredient backend.
package main

import (
	"log"
	"net/http"
	"os"

	"macros/internal/api"
	"macros/internal/utils"
)

func main() {
	recipes, ingredients, err := utils.LoadDataRecipe("data/ricette.json") //load data from json file
	if err != nil {
		log.Fatal(err)
	}
	proteinIngridients, carbIngridients, fatIngridients, err := utils.LoadDataIngridients("data/macro-sources.json")
	if err != nil {
		log.Fatal(err)
	}
	macroStore, err := utils.NewMacroStore("data/mydiet.json")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("loaded %d recipes, %d ingredients", len(recipes), len(ingredients))

	//start server logic
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	srv := api.New(recipes, ingredients, proteinIngridients, carbIngridients, fatIngridients, macroStore)
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Router()); err != nil {
		log.Fatal(err)
	}
}
