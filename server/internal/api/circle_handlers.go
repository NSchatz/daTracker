package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/auth"
)

func (s *Server) handleCreateCircle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	circle, err := s.circles.CreateCircle(r.Context(), req.Name, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create circle")
		return
	}

	writeJSON(w, http.StatusCreated, circle)
}

func (s *Server) handleJoinCircle(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	circleID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid circle id")
		return
	}

	var req struct {
		InviteCode string `json:"invite_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	circle, err := s.circles.GetCircleByInviteCode(r.Context(), req.InviteCode)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invite code")
		return
	}

	if circle.ID != circleID {
		writeError(w, http.StatusBadRequest, "invite code does not match circle")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	if err := s.circles.AddMember(r.Context(), circleID, userID, "member"); err != nil {
		writeError(w, http.StatusInternalServerError, "could not join circle")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "joined"})
}

func (s *Server) handleGetMembers(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	circleID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid circle id")
		return
	}

	members, err := s.circles.GetMembers(r.Context(), circleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not get members")
		return
	}

	writeJSON(w, http.StatusOK, members)
}

func (s *Server) handleGetUserCircles(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	circles, err := s.circles.GetUserCircles(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not get circles")
		return
	}

	writeJSON(w, http.StatusOK, circles)
}
