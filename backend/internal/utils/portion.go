package utils

import (
	"fmt"
	"math"
)

// PortionItem is one ingredient of a portion and how many grams of it to eat.
// Fixed items come with the quantity the recipe states; the others were computed.
type PortionItem struct {
	IngredientID string  `json:"ingredientId"`
	Name         string  `json:"name"`
	NameEN       string  `json:"nameEn"`
	Grams        float64 `json:"grams"`
	Fixed        bool    `json:"fixed"`
}

// Portion is a recipe scaled to a meal: the grams of each ingredient (adjustable
// ones first) and the kcal and macros they add up to.
type Portion struct {
	Items []PortionItem `json:"items"`
	Macros
}

// ComputePortion works out how much of each adjustable ingredient of r (those with no
// stated quantity) to eat so the whole recipe, fixed ingredients included, gets as
// close as possible to the target protein, carbs and fats.
//
// It is a non-negative least-squares fit in grams of macros: every subset of the
// adjustable ingredients is solved exactly (at most a few, so this is cheap) and the
// best solution with no negative amounts wins. Grams are then rounded for display
// (1 g under 50 g, 5 g above) and the totals are computed from the rounded values.
func ComputePortion(r Recipe, ingredients map[string]Ingredient, target Macros) (Portion, error) {
	var fixed, adjustable []PortionItem
	var cols [][3]float64 // macros per gram of each adjustable ingredient
	rest := [3]float64{target.Proteins, target.Carbs, target.Fats}

	for _, it := range r.Items {
		ing, ok := ingredients[it.IngredientID]
		if !ok {
			return Portion{}, fmt.Errorf("portion: recipe %s: unknown ingredient %s", r.ID, it.IngredientID)
		}
		item := PortionItem{IngredientID: ing.ID, Name: ing.Name, NameEN: ing.NameEN}
		if it.Grams == nil {
			adjustable = append(adjustable, item)
			cols = append(cols, [3]float64{ing.Protein / 100, ing.Carbs / 100, ing.Fats / 100})
			continue
		}
		item.Grams, item.Fixed = *it.Grams, true
		fixed = append(fixed, item)
		m := ing.For(*it.Grams)
		rest[0] -= m.Proteins
		rest[1] -= m.Carbs
		rest[2] -= m.Fats
	}

	for i, g := range solveNonNegative(cols, rest) {
		adjustable[i].Grams = roundGrams(g)
	}

	p := Portion{Items: append(adjustable, fixed...)}
	for _, it := range p.Items {
		m := ingredients[it.IngredientID].For(it.Grams)
		p.Kcal += m.Kcal
		p.Proteins += m.Proteins
		p.Carbs += m.Carbs
		p.Fats += m.Fats
	}
	p.Kcal = math.Round(p.Kcal)
	p.Proteins = math.Round(p.Proteins*10) / 10
	p.Carbs = math.Round(p.Carbs*10) / 10
	p.Fats = math.Round(p.Fats*10) / 10
	return p, nil
}

// solveNonNegative returns x >= 0 minimising |sum_j x[j]*cols[j] - b|.
func solveNonNegative(cols [][3]float64, b [3]float64) []float64 {
	n := len(cols)
	best := make([]float64, n) // all zero: error is |b|
	bestErr := residual(cols, best, b)
	for mask := 1; mask < 1<<n; mask++ {
		var idx []int
		for j := range n {
			if mask&(1<<j) != 0 {
				idx = append(idx, j)
			}
		}
		sub, ok := leastSquares(cols, idx, b)
		if !ok {
			continue
		}
		x := make([]float64, n)
		feasible := true
		for k, j := range idx {
			if sub[k] < 0 {
				feasible = false
				break
			}
			x[j] = sub[k]
		}
		if !feasible {
			continue
		}
		if e := residual(cols, x, b); e < bestErr-1e-9 {
			best, bestErr = x, e
		}
	}
	return best
}

// leastSquares solves the normal equations (AᵀA)x = Aᵀb for the columns idx by
// Gaussian elimination with partial pivoting; ok is false if they are singular.
func leastSquares(cols [][3]float64, idx []int, b [3]float64) (x []float64, ok bool) {
	k := len(idx)
	m := make([][]float64, k) // augmented matrix [AᵀA | Aᵀb]
	for r := range k {
		m[r] = make([]float64, k+1)
		for c := range k {
			m[r][c] = dot(cols[idx[r]], cols[idx[c]])
		}
		m[r][k] = dot(cols[idx[r]], b)
	}
	for c := range k {
		p := c
		for r := c + 1; r < k; r++ {
			if math.Abs(m[r][c]) > math.Abs(m[p][c]) {
				p = r
			}
		}
		if math.Abs(m[p][c]) < 1e-12 {
			return nil, false
		}
		m[c], m[p] = m[p], m[c]
		for r := range k {
			if r == c {
				continue
			}
			f := m[r][c] / m[c][c]
			for cc := c; cc <= k; cc++ {
				m[r][cc] -= f * m[c][cc]
			}
		}
	}
	x = make([]float64, k)
	for r := range k {
		x[r] = m[r][k] / m[r][r]
	}
	return x, true
}

func residual(cols [][3]float64, x []float64, b [3]float64) float64 {
	var e float64
	for i := range 3 {
		v := -b[i]
		for j, c := range cols {
			v += x[j] * c[i]
		}
		e += v * v
	}
	return e
}

func dot(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func roundGrams(g float64) float64 {
	if g < 50 {
		return math.Round(g)
	}
	return math.Round(g/5) * 5
}
