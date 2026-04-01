package api

import (
	"encoding/json"
	"net/http"

	"github.com/nschatz/tracker/server/internal/auth"
)

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	} `json:"user"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
		InviteCode  string `json:"invite_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Email == "" || req.DisplayName == "" || req.Password == "" || req.InviteCode == "" {
		writeError(w, http.StatusBadRequest, "email, display_name, password, and invite_code are required")
		return
	}

	circle, err := s.store.GetCircleByInviteCode(r.Context(), req.InviteCode)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invite code")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := s.store.CreateUser(r.Context(), req.Email, req.DisplayName, hash)
	if err != nil {
		writeError(w, http.StatusConflict, "could not create user")
		return
	}

	if err := s.store.AddMember(r.Context(), circle.ID, user.ID, "member"); err != nil {
		writeError(w, http.StatusInternalServerError, "could not add user to circle")
		return
	}

	token, err := s.auth.IssueToken(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	var resp authResponse
	resp.Token = token
	resp.User.ID = user.ID.String()
	resp.User.Email = user.Email
	resp.User.DisplayName = user.DisplayName

	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := s.auth.IssueToken(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	var resp authResponse
	resp.Token = token
	resp.User.ID = user.ID.String()
	resp.User.Email = user.Email
	resp.User.DisplayName = user.DisplayName

	writeJSON(w, http.StatusOK, resp)
}
