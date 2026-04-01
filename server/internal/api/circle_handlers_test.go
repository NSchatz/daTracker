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

type mockCircleStore struct {
	circles map[string]*model.Circle   // keyed by invite code
	byID    map[uuid.UUID]*model.Circle
	members []model.CircleMember
}

func newMockCircleStore() *mockCircleStore {
	return &mockCircleStore{
		circles: make(map[string]*model.Circle),
		byID:    make(map[uuid.UUID]*model.Circle),
	}
}

func (m *mockCircleStore) CreateCircle(_ context.Context, name string, createdBy uuid.UUID) (*model.Circle, error) {
	c := &model.Circle{
		ID:         uuid.New(),
		Name:       name,
		InviteCode: "testcode",
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
	}
	m.byID[c.ID] = c
	m.circles[c.InviteCode] = c
	return c, nil
}

func (m *mockCircleStore) GetUserCircles(_ context.Context, userID uuid.UUID) ([]model.Circle, error) {
	var result []model.Circle
	for _, c := range m.byID {
		if c.CreatedBy == userID {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (m *mockCircleStore) GetMembers(_ context.Context, circleID uuid.UUID) ([]model.CircleMember, error) {
	var result []model.CircleMember
	for _, mem := range m.members {
		if mem.CircleID == circleID {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockCircleStore) GetCircleByInviteCode(_ context.Context, code string) (*model.Circle, error) {
	c, ok := m.circles[code]
	if !ok {
		return nil, &notFoundError{code}
	}
	return c, nil
}

func (m *mockCircleStore) AddMember(_ context.Context, circleID, userID uuid.UUID, role string) error {
	m.members = append(m.members, model.CircleMember{
		CircleID: circleID,
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
	})
	return nil
}

func TestCreateCircle(t *testing.T) {
	authSvc := auth.New("test-secret")
	authStore := newMockStore()
	circleStore := newMockCircleStore()

	srv := NewServer(authSvc, authStore, circleStore, nil, nil)

	userID := uuid.New()
	token, err := authSvc.IssueToken(userID)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"name": "My Circle"})
	req := httptest.NewRequest(http.MethodPost, "/circles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var circle model.Circle
	if err := json.NewDecoder(rr.Body).Decode(&circle); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if circle.Name != "My Circle" {
		t.Errorf("expected name 'My Circle', got %q", circle.Name)
	}
	if circle.ID == uuid.Nil {
		t.Error("expected non-nil circle ID")
	}
}
