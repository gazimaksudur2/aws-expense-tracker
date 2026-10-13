package main_test

import (
	"testing"
	"time"

	"github.com/gazimaksudur2/aws-expense-tracker/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := auth.CheckPassword(hash, "correct-password"); err != nil {
		t.Error("correct password should pass")
	}
	if err := auth.CheckPassword(hash, "wrong-password"); err == nil {
		t.Error("wrong password should fail")
	}
}

func TestSignAndAuthorize(t *testing.T) {
	token, err := auth.SignToken("user-123", testSecret)
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	userID, err := auth.Authorize(token, testSecret)
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("expected user-123, got %s", userID)
	}
}

func TestAuthorize_WrongSecret(t *testing.T) {
	token, _ := auth.SignToken("user-123", testSecret)
	_, err := auth.Authorize(token, "different-secret")
	if err == nil {
		t.Error("expected error with wrong secret")
	}
}

func TestAuthorize_MissingToken(t *testing.T) {
	_, err := auth.Authorize("", testSecret)
	if err == nil {
		t.Error("expected error with empty token")
	}
}

func TestAuthorize_ExpiredToken(t *testing.T) {
	// Build an already-expired token manually
	claims := auth.Claims{
		UserID: "user-123",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("could not sign expired token: %v", err)
	}
	_, err = auth.Authorize(signed, testSecret)
	if err == nil {
		t.Error("expected error for expired token")
	}
}
