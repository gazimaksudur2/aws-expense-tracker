package main

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/akrylysov/algnhsa"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/analytics"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/auth"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/expense"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/id"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/receipts"
)

type contextKey string

const contextKeyUserID contextKey = "user_id"

const maxBodyBytes = 1 << 20 // 1 MiB

// server holds the dependencies every handler needs.
type server struct {
	users    auth.UserStore
	expenses expense.Store
	secret   string
	receipts *receipts.Presigner
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()

	// Local/Docker only: Cognito owns register/login on AWS (no API Gateway routes).
	if s.secret != "" {
		mux.HandleFunc("POST /users/register", s.handleRegister)
		mux.HandleFunc("POST /users/login", s.handleLogin)
	}

	// Protected routes
	mux.Handle("POST /expenses", s.authMiddleware(http.HandlerFunc(s.handleCreateExpense)))
	mux.Handle("GET /expenses/monthly", s.authMiddleware(http.HandlerFunc(s.handleMonthlyExpenses)))
	mux.Handle("POST /receipts/upload", s.authMiddleware(http.HandlerFunc(s.handleReceiptUpload)))
	mux.Handle("GET /analytics", s.authMiddleware(http.HandlerFunc(s.handleAnalytics)))
	return mux
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func userIDFrom(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(contextKeyUserID).(string)
	return uid, ok && uid != ""
}

func validMonth(m string) bool {
	_, err := time.Parse("2006-01", m)
	return err == nil
}

// ---- middleware ----

// authMiddleware sets contextKeyUserID from API Gateway authorizer context (Lambda)
// or from a local HS256 JWT (Docker / APP_MODE=http).
func (s *server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userID, ok := userIDFromAuthorizer(r.Context()); ok {
			ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if s.secret == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		header := r.Header.Get("Authorization")
		token, found := strings.CutPrefix(header, "Bearer ")
		if !found || token == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		userID, err := auth.Authorize(token, s.secret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// userIDFromAuthorizer reads Cognito JWT `sub` (phase 2) or legacy Lambda authorizer user_id.
func userIDFromAuthorizer(ctx context.Context) (string, bool) {
	event, ok := algnhsa.APIGatewayV2RequestFromContext(ctx)
	if !ok || event.RequestContext.Authorizer == nil {
		return "", false
	}
	authz := event.RequestContext.Authorizer
	if authz.JWT != nil {
		if sub := authz.JWT.Claims["sub"]; sub != "" {
			return sub, true
		}
	}
	if authz.Lambda != nil {
		if v, ok := authz.Lambda["user_id"].(string); ok && v != "" {
			return v, true
		}
	}
	return "", false
}

// ---- public handlers ----

func (s *server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	if !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "a valid email is required")
		return
	}
	if len(body.Password) < 8 || len(body.Password) > 72 { // bcrypt ignores/rejects > 72 bytes
		writeError(w, http.StatusBadRequest, "password must be 8-72 characters")
		return
	}

	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	user := auth.User{
		UserID:       "user_" + id.New(4),
		Email:        email,
		Name:         strings.TrimSpace(body.Name),
		PasswordHash: hash,
	}
	if err := s.users.CreateUser(user); err != nil {
		if errors.Is(err, auth.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"user_id": user.UserID})
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))

	// Same response for unknown email and wrong password: don't reveal which.
	user, err := s.users.GetUserByEmail(email)
	if err != nil || auth.CheckPassword(user.PasswordHash, body.Password) != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	token, err := auth.SignToken(user.UserID, s.secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// ---- protected handlers ----

func (s *server) handleCreateExpense(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Amount   string `json:"amount"`
		Category string `json:"category"`
		Date     string `json:"date"`
	}
	if !decodeBody(w, r, &body) {
		return
	}

	amount, err := strconv.ParseFloat(strings.TrimSpace(body.Amount), 64)
	if err != nil || amount <= 0 || math.IsInf(amount, 0) || math.IsNaN(amount) {
		writeError(w, http.StatusBadRequest, "amount must be a positive decimal string")
		return
	}
	category := strings.ToLower(strings.TrimSpace(body.Category))
	if category == "" {
		writeError(w, http.StatusBadRequest, "category is required")
		return
	}
	if _, err := time.Parse("2006-01-02", body.Date); err != nil {
		writeError(w, http.StatusBadRequest, "date must be YYYY-MM-DD")
		return
	}

	e := expense.Expense{
		ExpenseID: id.New(4),
		UserID:    userID, // always from the token, never from the body
		Amount:    strconv.FormatFloat(amount, 'f', 2, 64),
		Category:  category,
		Date:      body.Date,
	}
	if err := s.expenses.Put(e); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	e.UserID = "" // omitted from the response
	writeJSON(w, http.StatusCreated, e)
}

func (s *server) handleMonthlyExpenses(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	month := r.URL.Query().Get("month")
	if !validMonth(month) {
		writeError(w, http.StatusBadRequest, "month must be YYYY-MM")
		return
	}
	rows, err := s.expenses.QueryMonth(userID, month)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if rows == nil {
		rows = []expense.Expense{}
	}
	for i := range rows {
		rows[i].UserID = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{"month": month, "expenses": rows})
}

func (s *server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	month := r.URL.Query().Get("month")
	if !validMonth(month) {
		writeError(w, http.StatusBadRequest, "month must be YYYY-MM")
		return
	}
	rows, err := s.expenses.QueryMonth(userID, month)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	total, byCategory := analytics.Summarize(rows)
	if byCategory == nil {
		byCategory = []analytics.CategoryTotal{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"month":       month,
		"total":       total,
		"by_category": byCategory,
	})
}

// handleReceiptUpload returns a short-lived S3 presigned PUT URL.
// When RECEIPTS_BUCKET is unset, s.receipts is nil and this returns 501.
func (s *server) handleReceiptUpload(w http.ResponseWriter, r *http.Request) {
	if s.receipts == nil {
		writeError(w, http.StatusNotImplemented, "receipt upload is not configured")
		return
	}
	userID, ok := userIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		ExpenseID   string `json:"expense_id"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Date        string `json:"date"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if body.ExpenseID == "" || body.Filename == "" || body.ContentType == "" {
		writeError(w, http.StatusBadRequest, "expense_id, filename, and content_type are required")
		return
	}
	if _, err := time.Parse("2006-01-02", body.Date); err != nil {
		writeError(w, http.StatusBadRequest, "date must be YYYY-MM-DD")
		return
	}
	if !strings.HasPrefix(body.ContentType, "image/") && body.ContentType != "application/pdf" {
		writeError(w, http.StatusBadRequest, "content_type must start with image/* or application/pdf")
		return
	}

	e, err := s.expenses.Get(userID, body.Date, body.ExpenseID)
	if err != nil {
		if errors.Is(err, expense.ErrNotFound) {
			writeError(w, http.StatusNotFound, "expense not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	key := receipts.ObjectKey(userID, body.ExpenseID, body.Filename)
	url, expiresIn, err := s.receipts.PresignPut(r.Context(), key, body.ContentType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create upload URL")
		return
	}

	e.ReceiptKey = key
	if err := s.expenses.Put(e); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"upload_url":  url,
		"receipt_key": key,
		"expires_in":  expiresIn,
	})
}
