package main_test

import (
	"testing"

	"github.com/gazimaksudur2/aws-expense-tracker/internal/analytics"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/expense"
)

func TestSummarize_CategoryTotals(t *testing.T) {
	expenses := []expense.Expense{
		{UserID: "user-a", Amount: "42.50", Category: "groceries", Date: "2026-10-01"},
		{UserID: "user-a", Amount: "10.00", Category: "transport", Date: "2026-10-02"},
		{UserID: "user-a", Amount: "7.50", Category: "groceries", Date: "2026-10-03"},
	}

	total, byCategory := analytics.Summarize(expenses)

	if total != "60.00" {
		t.Errorf("expected grand total 60.00, got %s", total)
	}

	totals := map[string]string{}
	for _, c := range byCategory {
		totals[c.Category] = c.Total
	}

	if totals["groceries"] != "50.00" {
		t.Errorf("expected groceries=50.00, got %s", totals["groceries"])
	}
	if totals["transport"] != "10.00" {
		t.Errorf("expected transport=10.00, got %s", totals["transport"])
	}
}

func TestSummarize_Empty(t *testing.T) {
	total, byCategory := analytics.Summarize(nil)
	if total != "0.00" {
		t.Errorf("expected 0.00 for empty input, got %s", total)
	}
	if len(byCategory) != 0 {
		t.Errorf("expected empty categories, got %d", len(byCategory))
	}
}
