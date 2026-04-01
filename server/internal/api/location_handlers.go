package api

import (
	"encoding/json"
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

	w.WriteHeader(http.StatusAccepted)
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
