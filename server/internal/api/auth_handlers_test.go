package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/auth"
	"github.com/nschatz/tracker/server/internal/model"
)

type mockStore struct {
	users   map[string]*model.User
	circles map[string]*model.Circle
	members []struct {
		circleID uuid.UUID
		userID   uuid.UUID
		role     string
	}
}

func newMockStore() *mockStore {
	return &mockStore{
		users:   make(map[string]*model.User),
		circles: make(map[string]*model.Circle),
	}
}

func (m *mockStore) CreateUser(_ context.Context, email, displayName, passwordHash string) (*model.User, error) {
	u := &model.User{
		ID:           uuid.New(),
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	m.users[email] = u
	return u, nil
}

func (m *mockStore) GetUserByEmail(_ context.Context, email string) (*model.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, &notFoundError{email}
	}
	return u, nil
}

func (m *mockStore) GetCircleByInviteCode(_ context.Context, code string) (*model.Circle, error) {
	c, ok := m.circles[code]
	if !ok {
		return nil, &notFoundError{code}
	}
	return c, nil
}

func (m *mockStore) AddMember(_ context.Context, circleID, userID uuid.UUID, role string) error {
	m.members = append(m.members, struct {
		circleID uuid.UUID
		userID   uuid.UUID
		role     string
	}{circleID, userID, role})
	return nil
}

type notFoundError struct{ key string }

func (e *notFoundError) Error() string { return "not found: " + e.key }

func TestRegisterAndLogin(t *testing.T) {
	store := newMockStore()

	// Pre-create a circle with invite code "abc123"
	circleID := uuid.New()
	store.circles["abc123"] = &model.Circle{
		ID:         circleID,
		Name:       "Test Circle",
		InviteCode: "abc123",
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
	}

	a := auth.New("test-secret")
	srv := NewServer(a, store)

	// Register
	regBody, _ := json.Marshal(map[string]string{
		"email":        "alice@example.com",
		"display_name": "Alice",
		"password":     "hunter2",
		"invite_code":  "abc123",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var regResp authResponse
	if err := json.NewDecoder(rr.Body).Decode(&regResp); err != nil {
		t.Fatalf("register: decode response: %v", err)
	}
	if regResp.Token == "" {
		t.Fatal("register: expected non-empty token")
	}
	if regResp.User.ID == "" {
		t.Fatal("register: expected non-empty user id")
	}
	if regResp.User.Email != "alice@example.com" {
		t.Fatalf("register: expected email alice@example.com, got %s", regResp.User.Email)
	}
	if regResp.User.DisplayName != "Alice" {
		t.Fatalf("register: expected display_name Alice, got %s", regResp.User.DisplayName)
	}

	// Verify member was added to circle
	if len(store.members) != 1 {
		t.Fatalf("register: expected 1 circle member, got %d", len(store.members))
	}
	if store.members[0].circleID != circleID {
		t.Fatal("register: member added to wrong circle")
	}

	// Login
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "alice@example.com",
		"password": "hunter2",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	srv.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("login: want 200, got %d — body: %s", rr2.Code, rr2.Body.String())
	}

	var loginResp authResponse
	if err := json.NewDecoder(rr2.Body).Decode(&loginResp); err != nil {
		t.Fatalf("login: decode response: %v", err)
	}
	if loginResp.Token == "" {
		t.Fatal("login: expected non-empty token")
	}

	// Wrong password should fail
	badBody, _ := json.Marshal(map[string]string{
		"email":    "alice@example.com",
		"password": "wrongpassword",
	})
	req3 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(badBody))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	srv.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusUnauthorized {
		t.Fatalf("bad login: want 401, got %d", rr3.Code)
	}

	// Bad invite code should fail register
	badInvite, _ := json.Marshal(map[string]string{
		"email":        "bob@example.com",
		"display_name": "Bob",
		"password":     "password",
		"invite_code":  "invalid",
	})
	req4 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(badInvite))
	req4.Header.Set("Content-Type", "application/json")
	rr4 := httptest.NewRecorder()
	srv.ServeHTTP(rr4, req4)

	if rr4.Code != http.StatusBadRequest {
		t.Fatalf("bad invite: want 400, got %d", rr4.Code)
	}
}
