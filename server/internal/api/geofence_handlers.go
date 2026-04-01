package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/auth"
)

func (s *Server) handleCreateGeofence(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CircleID     uuid.UUID `json:"circle_id"`
		Name         string    `json:"name"`
		Lat          float64   `json:"lat"`
		Lng          float64   `json:"lng"`
		RadiusMeters float32   `json:"radius_meters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.RadiusMeters <= 0 {
		writeError(w, http.StatusBadRequest, "radius_meters must be greater than 0")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	gf, err := s.geofences.CreateGeofence(r.Context(), req.CircleID, req.Name, req.Lat, req.Lng, req.RadiusMeters, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create geofence")
		return
	}

	writeJSON(w, http.StatusCreated, gf)
}

func (s *Server) handleGetGeofences(w http.ResponseWriter, r *http.Request) {
	circleIDStr := r.URL.Query().Get("circle_id")
	if circleIDStr == "" {
		writeError(w, http.StatusBadRequest, "circle_id query parameter is required")
		return
	}
	circleID, err := uuid.Parse(circleIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid circle_id")
		return
	}

	geofences, err := s.geofences.GetGeofences(r.Context(), circleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not get geofences")
		return
	}

	writeJSON(w, http.StatusOK, geofences)
}

func (s *Server) handleUpdateGeofence(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid geofence id")
		return
	}

	var req struct {
		Name         string  `json:"name"`
		Lat          float64 `json:"lat"`
		Lng          float64 `json:"lng"`
		RadiusMeters float32 `json:"radius_meters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	gf, err := s.geofences.UpdateGeofence(r.Context(), id, req.Name, req.Lat, req.Lng, req.RadiusMeters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update geofence")
		return
	}

	writeJSON(w, http.StatusOK, gf)
}

func (s *Server) handleDeleteGeofence(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid geofence id")
		return
	}

	if err := s.geofences.DeleteGeofence(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete geofence")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
