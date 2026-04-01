package store_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/model"
	"github.com/nschatz/tracker/server/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://tracker:tracker@localhost:5432/tracker?sslmode=disable"
	}
	ctx := context.Background()
	s, err := store.New(ctx, url)
	if err != nil {
		t.Skipf("skipping integration test: cannot connect to database: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s+%s@example.com", prefix, uuid.New().String())
}

func TestCreateAndGetUser(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	email := uniqueEmail("testuser")
	displayName := "Test User"
	passwordHash := "$2a$10$fakehashfortest"

	created, err := s.CreateUser(ctx, email, displayName, passwordHash)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if created.Email != email {
		t.Errorf("expected email %q, got %q", email, created.Email)
	}
	if created.DisplayName != displayName {
		t.Errorf("expected display_name %q, got %q", displayName, created.DisplayName)
	}
	if created.PasswordHash != passwordHash {
		t.Errorf("expected password_hash to match")
	}
	if created.ID.String() == "" {
		t.Errorf("expected non-empty ID")
	}

	byEmail, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if byEmail.ID != created.ID {
		t.Errorf("GetUserByEmail: ID mismatch: got %v, want %v", byEmail.ID, created.ID)
	}

	byID, err := s.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if byID.Email != email {
		t.Errorf("GetUserByID: email mismatch: got %q, want %q", byID.Email, email)
	}
}

func TestCreateCircleAndJoin(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create circle owner
	owner, err := s.CreateUser(ctx, uniqueEmail("owner"), "Owner", "hash")
	if err != nil {
		t.Fatalf("CreateUser (owner): %v", err)
	}

	// Create circle
	circle, err := s.CreateCircle(ctx, "Test Circle", owner.ID)
	if err != nil {
		t.Fatalf("CreateCircle: %v", err)
	}
	if circle.Name != "Test Circle" {
		t.Errorf("expected circle name %q, got %q", "Test Circle", circle.Name)
	}
	if circle.InviteCode == "" {
		t.Error("expected non-empty invite code")
	}
	if len(circle.InviteCode) != 12 {
		t.Errorf("expected invite code length 12 (6 bytes hex), got %d", len(circle.InviteCode))
	}
	if circle.CreatedBy != owner.ID {
		t.Errorf("expected created_by %v, got %v", owner.ID, circle.CreatedBy)
	}

	// Owner should be an admin member already
	members, err := s.GetMembers(ctx, circle.ID)
	if err != nil {
		t.Fatalf("GetMembers: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member after create, got %d", len(members))
	}
	if members[0].UserID != owner.ID {
		t.Errorf("expected owner as first member")
	}
	if members[0].Role != "admin" {
		t.Errorf("expected role 'admin', got %q", members[0].Role)
	}

	// Create a second user and add them
	joiner, err := s.CreateUser(ctx, uniqueEmail("joiner"), "Joiner", "hash")
	if err != nil {
		t.Fatalf("CreateUser (joiner): %v", err)
	}

	if err := s.AddMember(ctx, circle.ID, joiner.ID, "member"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	// AddMember with ON CONFLICT DO NOTHING should be idempotent
	if err := s.AddMember(ctx, circle.ID, joiner.ID, "member"); err != nil {
		t.Fatalf("AddMember (duplicate) should not fail: %v", err)
	}

	members, err = s.GetMembers(ctx, circle.ID)
	if err != nil {
		t.Fatalf("GetMembers after join: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}

	// Look up circle by invite code
	found, err := s.GetCircleByInviteCode(ctx, circle.InviteCode)
	if err != nil {
		t.Fatalf("GetCircleByInviteCode: %v", err)
	}
	if found.ID != circle.ID {
		t.Errorf("GetCircleByInviteCode: ID mismatch")
	}

	// GetUserCircles for joiner should include the circle
	circles, err := s.GetUserCircles(ctx, joiner.ID)
	if err != nil {
		t.Fatalf("GetUserCircles: %v", err)
	}
	if len(circles) != 1 {
		t.Errorf("expected 1 circle for joiner, got %d", len(circles))
	}
	if circles[0].ID != circle.ID {
		t.Errorf("GetUserCircles: wrong circle returned")
	}
}

func TestInsertAndQueryLocations(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create owner and circle
	owner, err := s.CreateUser(ctx, uniqueEmail("loc-owner"), "Loc Owner", "hash")
	if err != nil {
		t.Fatalf("CreateUser (owner): %v", err)
	}
	circle, err := s.CreateCircle(ctx, "Loc Circle", owner.ID)
	if err != nil {
		t.Fatalf("CreateCircle: %v", err)
	}

	// Create a member and add them to the circle
	member, err := s.CreateUser(ctx, uniqueEmail("loc-member"), "Loc Member", "hash")
	if err != nil {
		t.Fatalf("CreateUser (member): %v", err)
	}
	if err := s.AddMember(ctx, circle.ID, member.ID, "member"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	// Insert 3 location points at different times
	now := time.Now().UTC().Truncate(time.Second)
	speed := float32(5.0)
	battery := int16(80)
	accuracy := float32(10.0)

	locs := []model.LocationInput{
		{Lat: 40.7128, Lng: -74.0060, Speed: &speed, BatteryLevel: &battery, Accuracy: &accuracy, RecordedAt: now.Add(-2 * time.Minute)},
		{Lat: 40.7130, Lng: -74.0058, RecordedAt: now.Add(-1 * time.Minute)},
		{Lat: 40.7135, Lng: -74.0055, RecordedAt: now},
	}

	if err := s.InsertLocations(ctx, member.ID, locs); err != nil {
		t.Fatalf("InsertLocations: %v", err)
	}

	// GetLatestLocations should return newest point for the member
	latest, err := s.GetLatestLocations(ctx, circle.ID)
	if err != nil {
		t.Fatalf("GetLatestLocations: %v", err)
	}

	// Find the member's entry in results
	var memberLoc *model.Location
	for i := range latest {
		if latest[i].UserID == member.ID {
			memberLoc = &latest[i]
			break
		}
	}
	if memberLoc == nil {
		t.Fatal("GetLatestLocations: member not found in results")
	}

	// Should be the newest point (index 2)
	if !memberLoc.RecordedAt.Equal(now) {
		t.Errorf("GetLatestLocations: expected newest point at %v, got %v", now, memberLoc.RecordedAt)
	}
	if memberLoc.UserID != member.ID {
		t.Errorf("GetLatestLocations: wrong user_id: got %v, want %v", memberLoc.UserID, member.ID)
	}

	// GetHistory should return all 3 points
	history, err := s.GetHistory(ctx, member.ID, now.Add(-10*time.Minute), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("GetHistory: expected 3 points, got %d", len(history))
	}
	// Verify ordered ASC
	if !history[0].RecordedAt.Before(history[1].RecordedAt) {
		t.Errorf("GetHistory: expected ascending order, got %v then %v", history[0].RecordedAt, history[1].RecordedAt)
	}
	if !history[1].RecordedAt.Before(history[2].RecordedAt) {
		t.Errorf("GetHistory: expected ascending order, got %v then %v", history[1].RecordedAt, history[2].RecordedAt)
	}
	// Verify all belong to the member
	for _, h := range history {
		if h.UserID != member.ID {
			t.Errorf("GetHistory: unexpected user_id %v", h.UserID)
		}
	}
}
