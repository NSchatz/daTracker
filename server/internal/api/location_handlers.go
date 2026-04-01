package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/auth"
	"github.com/nschatz/tracker/server/internal/model"
)

// handlePostLocations handles POST /locations
// Accepts {"locations": [...]} and returns 202 on success.
func (s *Server) handlePostLocations(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Locations []model.LocationInput `json:"locations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.Locations) == 0 {
		writeError(w, http.StatusBadRequest, "locations must not be empty")
		return
	}

	if err := s.locations.InsertLocations(r.Context(), userID, req.Locations); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store locations")
		return
	}

	// Get the latest point from the batch for broadcast and geofence eval
	latest := req.Locations[len(req.Locations)-1]
	loc := model.Location{
		UserID:       userID,
		Lat:          latest.Lat,
		Lng:          latest.Lng,
		Speed:        latest.Speed,
		BatteryLevel: latest.BatteryLevel,
		Accuracy:     latest.Accuracy,
		RecordedAt:   latest.RecordedAt,
	}

	// Run broadcast and geofence evaluation in a goroutine so it doesn't block the response
	go s.processLocationUpdate(userID, loc)

	w.WriteHeader(http.StatusAccepted)
}

// processLocationUpdate broadcasts the location and evaluates geofences.
// Uses a detached context since the request context will be cancelled.
func (s *Server) processLocationUpdate(userID uuid.UUID, loc model.Location) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.circles == nil {
		return
	}

	// Get user's circles
	circles, err := s.circles.GetUserCircles(ctx, userID)
	if err != nil {
		log.Printf("processLocationUpdate: get circles for user %s: %v", userID, err)
		return
	}

	for _, circle := range circles {
		// Broadcast location via WebSocket
		if s.hub != nil {
			s.hub.BroadcastLocation(circle.ID, loc)
		}

		// Evaluate geofences
		if s.geoEval == nil || s.geoTracker == nil || s.notifier == nil {
			continue
		}

		containingIDs, err := s.geoEval.FindContainingGeofences(ctx, circle.ID, loc.Lat, loc.Lng)
		if err != nil {
			log.Printf("processLocationUpdate: find geofences for circle %s: %v", circle.ID, err)
			continue
		}

		entered, left := s.geoTracker.Update(userID, containingIDs)
		if len(entered) == 0 && len(left) == 0 {
			continue
		}

		// Get user display name
		user, err := s.store.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("processLocationUpdate: get user %s: %v", userID, err)
			continue
		}

		// Get geofences for name lookup
		geofences, err := s.geoEval.GetGeofences(ctx, circle.ID)
		if err != nil {
			log.Printf("processLocationUpdate: get geofences for circle %s: %v", circle.ID, err)
			continue
		}
		geoMap := make(map[uuid.UUID]string, len(geofences))
		for _, g := range geofences {
			geoMap[g.ID] = g.Name
		}

		// Get member user IDs for notifications (excluding the current user)
		members, err := s.circles.GetMembers(ctx, circle.ID)
		if err != nil {
			log.Printf("processLocationUpdate: get members for circle %s: %v", circle.ID, err)
			continue
		}

		var memberIDs []string
		for _, m := range members {
			if m.UserID != userID {
				memberIDs = append(memberIDs, m.UserID.String())
			}
		}
		if len(memberIDs) == 0 {
			continue
		}

		for _, geoID := range entered {
			name := geoMap[geoID]
			s.notifier.GeofenceEnter(ctx, user.DisplayName, name, memberIDs)
		}
		for _, geoID := range left {
			name := geoMap[geoID]
			s.notifier.GeofenceLeave(ctx, user.DisplayName, name, memberIDs)
		}
	}
}

// handleGetLatestLocations handles GET /locations/latest?circle_id=UUID
// Returns the most recent location for each circle member.
func (s *Server) handleGetLatestLocations(w http.ResponseWriter, r *http.Request) {
	circleIDStr := r.URL.Query().Get("circle_id")
	if circleIDStr == "" {
		writeError(w, http.StatusBadRequest, "circle_id is required")
		return
	}
	circleID, err := uuid.Parse(circleIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid circle_id")
		return
	}

	locs, err := s.locations.GetLatestLocations(r.Context(), circleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get locations")
		return
	}

	writeJSON(w, http.StatusOK, locs)
}

// handleGetHistory handles GET /locations/history?user_id=UUID&from=RFC3339&to=RFC3339
// Returns location history for a user within the given time range.
func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		writeError(w, http.StatusBadRequest, "from and to are required")
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid from time: must be RFC3339")
		return
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid to time: must be RFC3339")
		return
	}

	locs, err := s.locations.GetHistory(r.Context(), userID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get history")
		return
	}

	writeJSON(w, http.StatusOK, locs)
}
