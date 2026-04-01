package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "supersecret"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if !CheckPassword(hash, password) {
		t.Error("CheckPassword returned false for correct password")
	}
	if CheckPassword(hash, "wrongpassword") {
		t.Error("CheckPassword returned true for wrong password")
	}
}

func TestIssueAndParseToken(t *testing.T) {
	a := New("test-secret")
	userID := uuid.New()

	token, err := a.IssueToken(userID)
	if err != nil {
		t.Fatalf("IssueToken error: %v", err)
	}

	parsed, err := a.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}

	if parsed != userID {
		t.Errorf("parsed UUID %v does not match original %v", parsed, userID)
	}
}

func TestParseTokenInvalid(t *testing.T) {
	a := New("test-secret")
	_, err := a.ParseToken("garbage")
	if err == nil {
		t.Error("expected error parsing invalid token, got nil")
	}
}

func TestMiddleware(t *testing.T) {
	a := New("test-secret")
	userID := uuid.New()

	token, err := a.IssueToken(userID)
	if err != nil {
		t.Fatalf("IssueToken error: %v", err)
	}

	var capturedID uuid.UUID
	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if capturedID != userID {
		t.Errorf("context UUID %v does not match original %v", capturedID, userID)
	}
}

func TestMiddlewareNoToken(t *testing.T) {
	a := New("test-secret")

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestMiddlewareTokenQueryParam(t *testing.T) {
	a := New("test-secret")
	userID := uuid.New()
	token, _ := a.IssueToken(userID)

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := UserIDFromContext(r.Context())
		if got != userID {
			t.Errorf("got = %v, want %v", got, userID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/ws?token="+token, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
