package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gazimaksudur2/aws-expense-tracker/internal/auth"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/expense"
)

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	// In-memory stores for now; Step 3 swaps these for DynamoDB.
	srv := &server{
		users:    auth.NewMemUserStore(),
		expenses: expense.NewMemStore(),
		secret:   secret,
	}

	switch os.Getenv("APP_MODE") {
	case "lambda":
		log.Fatal("APP_MODE=lambda is implemented in step 4")
	default: // "http"
		log.Println("listening on :8080")
		log.Fatal(http.ListenAndServe(":8080", srv.routes()))
	}
}
