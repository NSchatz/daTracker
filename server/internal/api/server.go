package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/auth"
	"github.com/nschatz/tracker/server/internal/model"
)

type AuthStore interface {
	CreateUser(ctx context.Context, email, displayName, passwordHash string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetCircleByInviteCode(ctx context.Context, code string) (*model.Circle, error)
	AddMember(ctx context.Context, circleID, userID uuid.UUID, role string) error
}

type Server struct {
	router chi.Router
	auth   *auth.Auth
	store  AuthStore
}

func NewServer(a *auth.Auth, store AuthStore) *Server {
	s := &Server{
		router: chi.NewRouter(),
		auth:   a,
		store:  store,
	}

	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})

	s.router.Post("/auth/register", s.handleRegister)
	s.router.Post("/auth/login", s.handleLogin)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
