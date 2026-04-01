package api

import (
	"encoding/json"
	"net/http"

	"github.com/nschatz/tracker/server/internal/auth"
)

func (s *Server) handleRegisterFCMToken(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	if err := s.fcmTokens.UpsertFCMToken(r.Context(), userID, req.Token); err != nil {
		writeError(w, http.StatusInternalServerError, "could not register FCM token")
		return
	}

	w.WriteHeader(http.StatusOK)
}
