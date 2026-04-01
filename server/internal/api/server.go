package api

import (
	"context"
	"io/fs"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/auth"
	"github.com/nschatz/tracker/server/internal/geo"
	"github.com/nschatz/tracker/server/internal/model"
	"github.com/nschatz/tracker/server/internal/notify"
	"github.com/nschatz/tracker/server/internal/ws"
)

type AuthStore interface {
	CreateUser(ctx context.Context, email, displayName, passwordHash string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type CircleStore interface {
	CreateCircle(ctx context.Context, name string, createdBy uuid.UUID) (*model.Circle, error)
	GetUserCircles(ctx context.Context, userID uuid.UUID) ([]model.Circle, error)
	GetMembers(ctx context.Context, circleID uuid.UUID) ([]model.CircleMember, error)
	GetCircleByInviteCode(ctx context.Context, code string) (*model.Circle, error)
	AddMember(ctx context.Context, circleID, userID uuid.UUID, role string) error
}

type LocationStore interface {
	InsertLocations(ctx context.Context, userID uuid.UUID, locs []model.LocationInput) error
	GetLatestLocations(ctx context.Context, circleID uuid.UUID) ([]model.Location, error)
	GetHistory(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]model.Location, error)
}

type GeofenceStore interface {
	CreateGeofence(ctx context.Context, circleID uuid.UUID, name string, lat, lng float64, radiusMeters float32, createdBy uuid.UUID) (*model.Geofence, error)
	GetGeofences(ctx context.Context, circleID uuid.UUID) ([]model.Geofence, error)
	UpdateGeofence(ctx context.Context, id uuid.UUID, name string, lat, lng float64, radiusMeters float32) (*model.Geofence, error)
	DeleteGeofence(ctx context.Context, id uuid.UUID) error
}

type GeoEvaluator interface {
	FindContainingGeofences(ctx context.Context, circleID uuid.UUID, lat, lng float64) ([]uuid.UUID, error)
	GetGeofences(ctx context.Context, circleID uuid.UUID) ([]model.Geofence, error)
}

type Server struct {
	router     chi.Router
	auth       *auth.Auth
	store      AuthStore
	circles    CircleStore
	locations  LocationStore
	geofences  GeofenceStore
	hub        *ws.Hub
	geoTracker *geo.Tracker
	notifier   *notify.Notifier
	geoEval    GeoEvaluator
}

func NewServer(a *auth.Auth, store AuthStore, circles CircleStore, locations LocationStore, geofences GeofenceStore, hub *ws.Hub, geoTracker *geo.Tracker, notifier *notify.Notifier, geoEval GeoEvaluator, webFS fs.FS) *Server {
	s := &Server{
		router:     chi.NewRouter(),
		auth:       a,
		store:      store,
		circles:    circles,
		locations:  locations,
		geofences:  geofences,
		hub:        hub,
		geoTracker: geoTracker,
		notifier:   notifier,
		geoEval:    geoEval,
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

		r.Post("/circles", s.handleCreateCircle)
		r.Post("/circles/{id}/join", s.handleJoinCircle)
		r.Get("/circles/{id}/members", s.handleGetMembers)
		r.Get("/circles", s.handleGetUserCircles)

		r.Post("/geofences", s.handleCreateGeofence)
		r.Get("/geofences", s.handleGetGeofences)
		r.Put("/geofences/{id}", s.handleUpdateGeofence)
		r.Delete("/geofences/{id}", s.handleDeleteGeofence)

		r.Get("/ws", s.handleWebSocket)
	})

	if webFS != nil {
		s.router.NotFound(staticFileHandler(webFS).ServeHTTP)
	}

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	circleIDStr := r.URL.Query().Get("circle_id")
	circleID, _ := uuid.Parse(circleIDStr)
	s.hub.HandleConnect(w, r, userID, circleID)
}
