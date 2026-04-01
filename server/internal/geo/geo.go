package geo

import (
	"sync"

	"github.com/google/uuid"
)

// Tracker maintains in-memory state of which geofences each user is currently inside.
type Tracker struct {
	mu    sync.Mutex
	state map[uuid.UUID]map[uuid.UUID]struct{} // user_id -> set of geofence_ids
}

func NewTracker() *Tracker {
	return &Tracker{
		state: make(map[uuid.UUID]map[uuid.UUID]struct{}),
	}
}

// Update takes the set of geofences a user is currently inside and returns
// which geofences were entered and which were left since the last update.
func (t *Tracker) Update(userID uuid.UUID, currentGeofences []uuid.UUID) (entered, left []uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	previous, ok := t.state[userID]
	if !ok {
		previous = make(map[uuid.UUID]struct{})
	}

	current := make(map[uuid.UUID]struct{}, len(currentGeofences))
	for _, id := range currentGeofences {
		current[id] = struct{}{}
	}

	// Find entered: in current but not in previous
	for id := range current {
		if _, wasThere := previous[id]; !wasThere {
			entered = append(entered, id)
		}
	}

	// Find left: in previous but not in current
	for id := range previous {
		if _, isThere := current[id]; !isThere {
			left = append(left, id)
		}
	}

	t.state[userID] = current
	return entered, left
}

// SetState rebuilds state for a user (used on server startup).
func (t *Tracker) SetState(userID uuid.UUID, geofenceIDs []uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	s := make(map[uuid.UUID]struct{}, len(geofenceIDs))
	for _, id := range geofenceIDs {
		s[id] = struct{}{}
	}
	t.state[userID] = s
}
