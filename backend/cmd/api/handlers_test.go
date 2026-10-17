package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/gazimaksudur2/aws-expense-tracker/internal/auth"
	"github.com/gazimaksudur2/aws-expense-tracker/internal/expense"
)

const testSecret = "handlers-test-secret"

func newTestHandler() http.Handler {
	s := &server{
		users:    auth.NewMemUserStore(),
		expenses: expense.NewMemStore(),
		secret:   testSecret,
	}
	return s.routes()
}

// do sends one request through the real router and decodes the JSON response.
func do(t *testing.T, h http.Handler, method, path, token, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var out map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s %s: response is not JSON: %q", method, path, rec.Body.String())
		}
	}
	return rec.Code, out
}

// registerAndLogin creates a user and returns a valid bearer token.
func registerAndLogin(t *testing.T, h http.Handler, email string) string {
	t.Helper()
	creds := `{"email":"` + email + `","name":"Test","password":"password123"}`
	if code, _ := do(t, h, "POST", "/users/register", "", creds); code != http.StatusCreated {
		t.Fatalf("register %s: got %d", email, code)
	}
	code, out := do(t, h, "POST", "/users/login", "", creds)
	if code != http.StatusOK {
		t.Fatalf("login %s: got %d", email, code)
	}
	token, _ := out["token"].(string)
	if token == "" {
		t.Fatal("login returned no token")
	}
	return token
}

func TestRegister_DuplicateEmail(t *testing.T) {
	h := newTestHandler()
	body := `{"email":"a@test.com","name":"A","password":"password123"}`
	if code, _ := do(t, h, "POST", "/users/register", "", body); code != http.StatusCreated {
		t.Fatalf("first register: got %d", code)
	}
	// Same email with different casing must still conflict.
	dup := `{"email":"A@Test.com","name":"A","password":"password123"}`
	if code, _ := do(t, h, "POST", "/users/register", "", dup); code != http.StatusConflict {
		t.Errorf("duplicate register: expected 409, got %d", code)
	}
}

func TestRegister_Validation(t *testing.T) {
	h := newTestHandler()
	cases := map[string]string{
		"short password": `{"email":"a@test.com","name":"A","password":"short"}`,
		"bad email":      `{"email":"not-an-email","name":"A","password":"password123"}`,
		"not json":       `nope`,
	}
	for name, body := range cases {
		if code, _ := do(t, h, "POST", "/users/register", "", body); code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, code)
		}
	}
}

func TestLogin_BadCredentials(t *testing.T) {
	h := newTestHandler()
	registerAndLogin(t, h, "a@test.com")

	wrongPass := `{"email":"a@test.com","password":"wrong-password"}`
	unknown := `{"email":"nobody@test.com","password":"password123"}`

	code1, out1 := do(t, h, "POST", "/users/login", "", wrongPass)
	code2, out2 := do(t, h, "POST", "/users/login", "", unknown)
	if code1 != http.StatusUnauthorized || code2 != http.StatusUnauthorized {
		t.Fatalf("expected 401 for both, got %d and %d", code1, code2)
	}
	// Must not reveal whether the email exists.
	if out1["error"] != out2["error"] {
		t.Errorf("error messages differ: %v vs %v", out1["error"], out2["error"])
	}
}

func TestProtectedRoutes_RejectMissingAndInvalidTokens(t *testing.T) {
	h := newTestHandler()

	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		UserID: "user_x",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})
	expiredToken, err := expired.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	wrongSecret, _ := auth.SignToken("user_x", "some-other-secret")

	tokens := map[string]string{
		"missing token":   "",
		"garbage token":   "garbage.token.here",
		"expired token":   expiredToken,
		"wrong signature": wrongSecret,
	}
	routes := []struct{ method, path string }{
		{"POST", "/expenses"},
		{"GET", "/expenses/monthly?month=2026-10"},
		{"GET", "/analytics?month=2026-10"},
		{"POST", "/receipts/upload"},
	}
	for name, tok := range tokens {
		for _, r := range routes {
			if code, _ := do(t, h, r.method, r.path, tok, `{}`); code != http.StatusUnauthorized {
				t.Errorf("%s on %s %s: expected 401, got %d", name, r.method, r.path, code)
			}
		}
	}
}

func TestCreateExpense_Validation(t *testing.T) {
	h := newTestHandler()
	tok := registerAndLogin(t, h, "a@test.com")

	cases := map[string]string{
		"negative amount":  `{"amount":"-5","category":"food","date":"2026-10-10"}`,
		"non-numeric":      `{"amount":"abc","category":"food","date":"2026-10-10"}`,
		"missing category": `{"amount":"5","category":"  ","date":"2026-10-10"}`,
		"bad date":         `{"amount":"5","category":"food","date":"10/10/2026"}`,
		"not json":         `nope`,
	}
	for name, body := range cases {
		if code, _ := do(t, h, "POST", "/expenses", tok, body); code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, code)
		}
	}
}

func TestCreateExpense_NormalizesAndIgnoresBodyUserID(t *testing.T) {
	h := newTestHandler()
	tokA := registerAndLogin(t, h, "a@test.com")
	tokB := registerAndLogin(t, h, "b@test.com")

	// Caller tries to write into someone else's partition via the body.
	body := `{"user_id":"someone-else","amount":"42.5","category":" Groceries ","date":"2026-10-10"}`
	code, out := do(t, h, "POST", "/expenses", tokA, body)
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", code)
	}
	if out["amount"] != "42.50" {
		t.Errorf("amount: expected 42.50, got %v", out["amount"])
	}
	if out["category"] != "groceries" {
		t.Errorf("category: expected groceries, got %v", out["category"])
	}
	if id, _ := out["expense_id"].(string); id == "" {
		t.Error("expected an expense_id")
	}

	// It must land under the caller, not under the body's user_id.
	_, mine := do(t, h, "GET", "/expenses/monthly?month=2026-10", tokA, "")
	if rows, _ := mine["expenses"].([]any); len(rows) != 1 {
		t.Errorf("caller should see 1 expense, got %v", mine["expenses"])
	}
	_, theirs := do(t, h, "GET", "/expenses/monthly?month=2026-10", tokB, "")
	if rows, _ := theirs["expenses"].([]any); len(rows) != 0 {
		t.Errorf("other user must see 0 expenses, got %v", theirs["expenses"])
	}
}

func TestMonthlyExpenses(t *testing.T) {
	h := newTestHandler()
	tok := registerAndLogin(t, h, "a@test.com")

	do(t, h, "POST", "/expenses", tok, `{"amount":"10","category":"transport","date":"2026-10-12"}`)
	do(t, h, "POST", "/expenses", tok, `{"amount":"42.5","category":"groceries","date":"2026-10-10"}`)
	do(t, h, "POST", "/expenses", tok, `{"amount":"99","category":"rent","date":"2026-09-01"}`)

	code, out := do(t, h, "GET", "/expenses/monthly?month=2026-10", tok, "")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if out["month"] != "2026-10" {
		t.Errorf("month: got %v", out["month"])
	}
	rows, _ := out["expenses"].([]any)
	if len(rows) != 2 {
		t.Fatalf("expected 2 October rows, got %d", len(rows))
	}
	// Ordered by date, like the DynamoDB sort key.
	first, _ := rows[0].(map[string]any)
	if first["date"] != "2026-10-10" {
		t.Errorf("rows not ordered by date, first is %v", first["date"])
	}
	if _, leaked := first["user_id"]; leaked {
		t.Error("response must not include user_id")
	}
}

func TestMonthlyExpenses_EmptyMonthIsEmptyArray(t *testing.T) {
	h := newTestHandler()
	tok := registerAndLogin(t, h, "a@test.com")

	code, out := do(t, h, "GET", "/expenses/monthly?month=2026-11", tok, "")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	rows, ok := out["expenses"].([]any)
	if !ok || len(rows) != 0 {
		t.Errorf("expected [] (not null), got %v", out["expenses"])
	}
}

func TestMonthParam_Validation(t *testing.T) {
	h := newTestHandler()
	tok := registerAndLogin(t, h, "a@test.com")

	for _, path := range []string{
		"/expenses/monthly",
		"/expenses/monthly?month=oct",
		"/expenses/monthly?month=2026-13",
		"/analytics",
		"/analytics?month=2026",
	} {
		if code, _ := do(t, h, "GET", path, tok, ""); code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", path, code)
		}
	}
}

func TestAnalytics(t *testing.T) {
	h := newTestHandler()
	tok := registerAndLogin(t, h, "a@test.com")

	do(t, h, "POST", "/expenses", tok, `{"amount":"42.5","category":"groceries","date":"2026-10-10"}`)
	do(t, h, "POST", "/expenses", tok, `{"amount":"7.5","category":"groceries","date":"2026-10-11"}`)
	do(t, h, "POST", "/expenses", tok, `{"amount":"10","category":"transport","date":"2026-10-12"}`)

	code, out := do(t, h, "GET", "/analytics?month=2026-10", tok, "")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if out["total"] != "60.00" {
		t.Errorf("total: expected 60.00, got %v", out["total"])
	}
	cats, _ := out["by_category"].([]any)
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %v", out["by_category"])
	}
	got := map[string]string{}
	for _, c := range cats {
		m, _ := c.(map[string]any)
		got[m["category"].(string)] = m["total"].(string)
	}
	if got["groceries"] != "50.00" || got["transport"] != "10.00" {
		t.Errorf("unexpected category totals: %v", got)
	}
}

func TestAnalytics_EmptyMonth(t *testing.T) {
	h := newTestHandler()
	tok := registerAndLogin(t, h, "a@test.com")

	code, out := do(t, h, "GET", "/analytics?month=2026-11", tok, "")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if out["total"] != "0.00" {
		t.Errorf("total: expected 0.00, got %v", out["total"])
	}
	if cats, ok := out["by_category"].([]any); !ok || len(cats) != 0 {
		t.Errorf("expected [] (not null), got %v", out["by_category"])
	}
}

func TestAnalytics_UserIsolation(t *testing.T) {
	h := newTestHandler()
	tokA := registerAndLogin(t, h, "a@test.com")
	tokB := registerAndLogin(t, h, "b@test.com")

	do(t, h, "POST", "/expenses", tokA, `{"amount":"100","category":"rent","date":"2026-10-01"}`)

	_, out := do(t, h, "GET", "/analytics?month=2026-10", tokB, "")
	if out["total"] != "0.00" {
		t.Errorf("user B must not see user A's totals, got %v", out["total"])
	}
}

func TestReceiptUpload_NotImplementedYet(t *testing.T) {
	h := newTestHandler()
	tok := registerAndLogin(t, h, "a@test.com")

	if code, _ := do(t, h, "POST", "/receipts/upload", tok, `{}`); code != http.StatusNotImplemented {
		t.Errorf("expected 501 until step 7, got %d", code)
	}
}
