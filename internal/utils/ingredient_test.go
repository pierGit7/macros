package utils_test

import (
	"testing"

	"macros/internal/utils"
)

const macroSourcesPath = "../../data/macro-sources.json"

func TestLoadDataIngridients(t *testing.T) {
	protein, carb, fat, err := utils.LoadDataIngridients(macroSourcesPath)
	if err != nil {
		t.Fatalf("LoadDataIngridients: %v", err)
	}

	for _, tc := range []struct {
		name string
		got  int
		want int
	}{
		{"protein", len(protein), 99},
		{"carb", len(carb), 152},
		{"fat", len(fat), 44},
	} {
		if tc.got != tc.want {
			t.Errorf("%s source count = %d, want %d", tc.name, tc.got, tc.want)
		}
	}

	// every ingredient sits in exactly one list
	seen := make(map[string]int)
	for _, m := range []map[string]utils.Ingredient{protein, carb, fat} {
		for id := range m {
			seen[id]++
		}
	}
	if len(seen) != 295 {
		t.Errorf("distinct ingredients across lists = %d, want 295", len(seen))
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("ingredient %s appears in %d lists, want 1", id, n)
		}
	}

	// a known lean protein source is keyed by its ID and carries its macros
	const tunaID = "664879ef3f91ab24e2996d18"
	tuna, ok := protein[tunaID]
	if !ok {
		t.Fatalf("protein map missing %s (Trancio di tonno)", tunaID)
	}
	if tuna.Name != "Trancio di tonno" {
		t.Errorf("protein[%s].Name = %q, want %q", tunaID, tuna.Name, "Trancio di tonno")
	}
	if tuna.Protein <= 0 {
		t.Errorf("protein[%s].Protein = %v, want > 0", tunaID, tuna.Protein)
	}
}

func TestLoadDataIngridientsMissingFile(t *testing.T) {
	if _, _, _, err := utils.LoadDataIngridients("does-not-exist.json"); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
