package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/akrylysov/algnhsa"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/auth"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/expense"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/store"
)

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	users, expenses := buildStores()
	srv := &server{
		users:    users,
		expenses: expenses,
		secret:   secret,
	}

	switch os.Getenv("APP_MODE") {
	case "lambda":
		algnhsa.ListenAndServe(srv.routes(), nil)
	default: // "http"
		log.Println("listening on :8080")
		log.Fatal(http.ListenAndServe(":8080", srv.routes()))
	}
}

func buildStores() (auth.UserStore, expense.Store) {
	usersTable := os.Getenv("USERS_TABLE")
	expensesTable := os.Getenv("EXPENSES_TABLE")
	if usersTable == "" || expensesTable == "" {
		log.Println("USERS_TABLE and EXPENSES_TABLE must be set: using in-memory stores")
		return auth.NewMemUserStore(), expense.NewMemStore()
	}

	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	ds, err := store.NewDynamo(context.Background(), endpoint, usersTable, expensesTable)
	if err != nil {
		log.Fatalf("dynamodb client: %v", err)
	}
	if endpoint != "" {
		ensureTablesWithRetry(ds)
	}
	log.Printf("using DynamoDB (users=%s, expenses=%s endpoint=%q)", usersTable, expensesTable, endpoint)
	return ds, ds
}

func ensureTablesWithRetry(ds *store.DynamoStore) {
	var err error
	const maxAttempts = 30
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = ds.EnsureTables(ctx)
		cancel()

		if err == nil {
			log.Println("dynamodb tables created successfully")
			return
		}
		log.Printf("waiting for DynamoDB tables (%d/%d): %v", attempt, maxAttempts, err)
		time.Sleep(time.Second)
	}
	log.Fatalf("dynamodb not reachable: %v", err)
}
