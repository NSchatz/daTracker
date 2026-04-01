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

type mockLocationStore struct {
	inserted []model.LocationInput
	forUser  uuid.UUID
}

func (m *mockLocationStore) InsertLocations(_ context.Context, userID uuid.UUID, locs []model.LocationInput) error {
	m.forUser = userID
	m.inserted = append(m.inserted, locs...)
	return nil
}

func (m *mockLocationStore) GetLatestLocations(_ context.Context, circleID uuid.UUID) ([]model.Location, error) {
	return []model.Location{}, nil
}

func (m *mockLocationStore) GetHistory(_ context.Context, userID uuid.UUID, from, to time.Time) ([]model.Location, error) {
	return []model.Location{}, nil
}

func TestPostLocations(t *testing.T) {
	authSvc := auth.New("test-secret")
	authStore := newMockStore()
	locStore := &mockLocationStore{}

	srv := NewServer(authSvc, authStore, locStore)

	// Create a user and issue a token
	userID := uuid.New()
	token, err := authSvc.IssueToken(userID)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	speed := float32(3.5)
	locs := []model.LocationInput{
		{Lat: 40.7128, Lng: -74.0060, Speed: &speed, RecordedAt: now.Add(-time.Minute)},
		{Lat: 40.7130, Lng: -74.0058, RecordedAt: now},
	}

	body, _ := json.Marshal(map[string]any{
		"locations": locs,
	})

	req := httptest.NewRequest(http.MethodPost, "/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("want 202, got %d — body: %s", rr.Code, rr.Body.String())
	}

	// Verify locations were stored
	if len(locStore.inserted) != 2 {
		t.Fatalf("expected 2 inserted locations, got %d", len(locStore.inserted))
	}
	if locStore.forUser != userID {
		t.Errorf("expected userID %v, got %v", userID, locStore.forUser)
	}
	if locStore.inserted[0].Lat != 40.7128 {
		t.Errorf("expected lat 40.7128, got %v", locStore.inserted[0].Lat)
	}
}

func TestPostLocations_Unauthenticated(t *testing.T) {
	authSvc := auth.New("test-secret")
	authStore := newMockStore()
	locStore := &mockLocationStore{}

	srv := NewServer(authSvc, authStore, locStore)

	body, _ := json.Marshal(map[string]any{
		"locations": []model.LocationInput{{Lat: 1.0, Lng: 1.0, RecordedAt: time.Now()}},
	})

	req := httptest.NewRequest(http.MethodPost, "/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestGetLatestLocations(t *testing.T) {
	authSvc := auth.New("test-secret")
	authStore := newMockStore()
	locStore := &mockLocationStore{}

	srv := NewServer(authSvc, authStore, locStore)

	userID := uuid.New()
	token, err := authSvc.IssueToken(userID)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	circleID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/locations/latest?circle_id="+circleID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestGetHistory(t *testing.T) {
	authSvc := auth.New("test-secret")
	authStore := newMockStore()
	locStore := &mockLocationStore{}

	srv := NewServer(authSvc, authStore, locStore)

	userID := uuid.New()
	token, err := authSvc.IssueToken(userID)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	from := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	to := time.Now().UTC().Format(time.RFC3339)
	url := "/locations/history?user_id=" + userID.String() + "&from=" + from + "&to=" + to

	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
}
