package analytics

import (
	"sort"
	"strconv"

	"github.com/gazimaksudur2/aws-expense-tracker/internal/expense"
)

type CategoryTotal struct {
	Category string `json:"category"`
	Total    string `json:"total"`
}

// Summarize returns the grand total and per-category totals (sorted by category).
func Summarize(expenses []expense.Expense) (total string, byCategory []CategoryTotal) {
	sums := map[string]float64{}
	grand := 0.0
	for _, e := range expenses {
		v, _ := strconv.ParseFloat(e.Amount, 64)
		sums[e.Category] += v
		grand += v
	}

	for cat, v := range sums {
		byCategory = append(byCategory, CategoryTotal{
			Category: cat,
			Total:    strconv.FormatFloat(v, 'f', 2, 64),
		})
	}
	sort.Slice(byCategory, func(i, j int) bool { return byCategory[i].Category < byCategory[j].Category })
	total = strconv.FormatFloat(grand, 'f', 2, 64)
	return
}
