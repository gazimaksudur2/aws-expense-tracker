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
	"github.com/gazimaksudur2/aws-expense-tracker/internal/receipts"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/store"
)

func main() {
	mode := os.Getenv("APP_MODE")
	secret := os.Getenv("JWT_SECRET")
	// Local/Docker still signs HS256 tokens. Lambda trusts API Gateway Cognito JWT authorizer.
	if mode != "lambda" && secret == "" {
		log.Fatal("JWT_SECRET must be set when APP_MODE is not lambda")
	}

	users, expenses := buildStores()
	var presigner *receipts.Presigner
	if bucket := os.Getenv("RECEIPTS_BUCKET"); bucket != "" {
		p, err := receipts.NewPresigner(context.Background(), bucket, os.Getenv("S3_ENDPOINT"))
		if err != nil {
			log.Fatalf("failed to create presigner: %v", err)
		}
		presigner = p
		log.Printf("receipt uploads enabled (bucket=%s endpoint=%q)", bucket, os.Getenv("S3_ENDPOINT"))
	} else {
		log.Println("RECEIPTS_BUCKET is not set: POST /receipts/upload returns 501")
	}

	srv := &server{
		users:    users,
		expenses: expenses,
		secret:   secret,
		receipts: presigner,
	}

	switch mode {
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

	if expensesTable == "" {
		log.Println("EXPENSES_TABLE unset: using in-memory stores")
		return auth.NewMemUserStore(), expense.NewMemStore()
	}

	// DynamoStore constructor needs a users table name; Cognito Lambda leaves USERS_TABLE unset.
	dynamoUsers := usersTable
	if dynamoUsers == "" {
		dynamoUsers = "users"
	}

	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	ds, err := store.NewDynamo(context.Background(), endpoint, dynamoUsers, expensesTable)
	if err != nil {
		log.Fatalf("dynamodb client: %v", err)
	}
	if endpoint != "" {
		ensureTablesWithRetry(ds)
	}

	var users auth.UserStore = ds
	if usersTable == "" {
		log.Println("USERS_TABLE unset: Cognito owns identity; using in-memory user store")
		users = auth.NewMemUserStore()
	}
	log.Printf("using DynamoDB (users=%s, expenses=%s endpoint=%q)", dynamoUsers, expensesTable, endpoint)
	return users, ds
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
