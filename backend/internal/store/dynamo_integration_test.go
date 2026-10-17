//go:build integration

package store_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gazimaksudur2/aws-expense-tracker/internal/auth"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/expense"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/id"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/store"
)

func newStore(t *testing.T) *store.DynamoStore {
	t.Helper()
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8000"
	}
	suffix := id.New(3)
	s, err := store.NewDynamo(context.Background(), endpoint, "users-"+suffix, "expenses-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.EnsureTables(ctx); err != nil {
		t.Fatalf("EnsureTables (is DynamoDB Local running on %s?): %v", endpoint, err)
	}
	return s
}

func TestUsers_CreateGetAndDuplicate(t *testing.T) {
	s := newStore(t)
	u := auth.User{UserID: "user_1", Email: "a@test.com", Name: "A", PasswordHash: "hash"}

	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := s.CreateUser(u); !errors.Is(err, auth.ErrEmailTaken) {
		t.Errorf("Duplicate CreateUser: expected ErrEmailTaken, got %v", err)
	}

	got, err := s.GetUserByEmail("a@test.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got.UserID != "user_1" || got.Email != "a@test.com" || got.Name != "A" || got.PasswordHash != "hash" {
		t.Errorf("unexpected user: %+v", got)
	}

	if _, err := s.GetUserByEmail("nobody@test.com"); !errors.Is(err, auth.ErrUserNotFound) {
		t.Errorf("missing user: expected ErrUserNotFound, got %v", err)
	}
}

func TestExpenses_MonthQueryIsolation(t *testing.T) {
	s := newStore(t)
	put := func(user, date, amount, cat string) {
		t.Helper()
		if err := s.Put(expense.Expense{UserID: user, Date: date, Amount: amount, Category: cat}); err != nil {
			t.Fatal(err)
		}
	}
	put("user_a", "2026-10-12", "10.0", "transportation")
	put("user_a", "2026-10-13", "20.0", "food")
	put("user_b", "2026-10-12", "30.0", "transportation")
	put("user_b", "2026-10-13", "40.0", "food")

	rows, err := s.QueryMonth("user_a", "2026-10")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 October rows for user_a, got %d: %+v", len(rows), rows)
	}
	if rows[0].Date != "2026-10-12" || rows[1].Date != "2026-10-13" {
		t.Errorf("rows not in sort key order: %+v", rows)
	}
	if rows[0].Amount != "10.0" || rows[1].Amount != "20.0" {
		t.Errorf("rows not in sort key order: %+v", rows)
	}
	if rows[0].Category != "transportation" || rows[1].Category != "food" {
		t.Errorf("rows not in sort key order: %+v", rows)
	}
	for _, r := range rows {
		if r.UserID != "user_a" {
			t.Errorf("leaked another user's row: %+v", r)
		}
		if r.ExpenseID == "" {
			t.Error("expected a generated expense ID")
		}
	}

	empty, err := s.QueryMonth("user_a", "2026-11")
	if err != nil || len(empty) != 0 {
		t.Errorf("expected no rows for user_a in November, got %d: %+v", len(empty), empty)
	}
}
