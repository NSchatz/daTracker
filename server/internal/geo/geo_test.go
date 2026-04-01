package geo

import (
	"testing"

	"github.com/google/uuid"
)

func TestDetectTransitions(t *testing.T) {
	tracker := NewTracker()
	userID := uuid.New()
	gfHome := uuid.New()
	gfWork := uuid.New()

	// Step 1: User starts nowhere -> enters gfHome
	entered, left := tracker.Update(userID, []uuid.UUID{gfHome})
	if len(entered) != 1 || entered[0] != gfHome {
		t.Fatalf("step 1: expected entered=[gfHome], got %v", entered)
	}
	if len(left) != 0 {
		t.Fatalf("step 1: expected left=[], got %v", left)
	}

	// Step 2: User moves to gfWork -> entered=[gfWork], left=[gfHome]
	entered, left = tracker.Update(userID, []uuid.UUID{gfWork})
	if len(entered) != 1 || entered[0] != gfWork {
		t.Fatalf("step 2: expected entered=[gfWork], got %v", entered)
	}
	if len(left) != 1 || left[0] != gfHome {
		t.Fatalf("step 2: expected left=[gfHome], got %v", left)
	}

	// Step 3: User stays at gfWork -> entered=[], left=[]
	entered, left = tracker.Update(userID, []uuid.UUID{gfWork})
	if len(entered) != 0 {
		t.Fatalf("step 3: expected entered=[], got %v", entered)
	}
	if len(left) != 0 {
		t.Fatalf("step 3: expected left=[], got %v", left)
	}

	// Step 4: User leaves all -> entered=[], left=[gfWork]
	entered, left = tracker.Update(userID, []uuid.UUID{})
	if len(entered) != 0 {
		t.Fatalf("step 4: expected entered=[], got %v", entered)
	}
	if len(left) != 1 || left[0] != gfWork {
		t.Fatalf("step 4: expected left=[gfWork], got %v", left)
	}
}
