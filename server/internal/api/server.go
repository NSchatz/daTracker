package api

import (
	"context"
	"net/http"
	"time"

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

type LocationStore interface {
	InsertLocations(ctx context.Context, userID uuid.UUID, locs []model.LocationInput) error
	GetLatestLocations(ctx context.Context, circleID uuid.UUID) ([]model.Location, error)
	GetHistory(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]model.Location, error)
}

type Server struct {
	router    chi.Router
	auth      *auth.Auth
	store     AuthStore
	locations LocationStore
}

func NewServer(a *auth.Auth, store AuthStore, locations LocationStore) *Server {
	s := &Server{
		router:    chi.NewRouter(),
		auth:      a,
		store:     store,
		locations: locations,
	}

	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})

	s.router.Post("/auth/register", s.handleRegister)
	s.router.Post("/auth/login", s.handleLogin)

	s.router.Group(func(r chi.Router) {
		r.Use(a.Middleware)
		r.Post("/locations", s.handlePostLocations)
		r.Get("/locations/latest", s.handleGetLatestLocations)
		r.Get("/locations/history", s.handleGetHistory)
	})

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
