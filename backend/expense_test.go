package main_test

import (
	"testing"

	"github.com/gazimaksudur2/aws-expense-tracker/internal/expense"
)

func TestPutAndQueryMonth(t *testing.T) {
	store := expense.NewMemStore()

	_ = store.Put(expense.Expense{
		UserID:   "user-a",
		Amount:   "42.50",
		Category: "groceries",
		Date:     "2026-10-10",
	})
	_ = store.Put(expense.Expense{
		UserID:   "user-a",
		Amount:   "10.00",
		Category: "transport",
		Date:     "2026-10-15",
	})
	// Different month — must not appear in October query
	_ = store.Put(expense.Expense{
		UserID:   "user-a",
		Amount:   "99.99",
		Category: "rent",
		Date:     "2026-09-01",
	})

	rows, err := store.QueryMonth("user-a", "2026-10")
	if err != nil {
		t.Fatalf("QueryMonth: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows for October, got %d", len(rows))
	}
}

func TestUserIsolation(t *testing.T) {
	store := expense.NewMemStore()

	_ = store.Put(expense.Expense{UserID: "user-a", Amount: "10.00", Category: "food", Date: "2026-10-01"})
	_ = store.Put(expense.Expense{UserID: "user-b", Amount: "20.00", Category: "food", Date: "2026-10-01"})

	rows, err := store.QueryMonth("user-a", "2026-10")
	if err != nil {
		t.Fatalf("QueryMonth: %v", err)
	}
	for _, r := range rows {
		if r.UserID != "user-a" {
			t.Errorf("user-a query returned a row belonging to %s", r.UserID)
		}
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row for user-a, got %d", len(rows))
	}
}
