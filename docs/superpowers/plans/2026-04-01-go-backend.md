# Go Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go API server with PostGIS storage for real-time location sharing, geofence alerts, and location history.

**Architecture:** Single Go binary serving REST + WebSocket endpoints, backed by PostgreSQL/PostGIS. Handlers depend on store interfaces for testability. Location ingestion triggers geofence evaluation and WebSocket broadcast. FCM used for push notifications when clients are disconnected.

**Tech Stack:** Go 1.22+, chi router, pgx (Postgres driver), coder/websocket, golang-jwt/jwt/v5, firebase-admin-go, PostGIS, Docker Compose

---

## File Structure

```
server/
  cmd/tracker/main.go              - entrypoint, wires all components
  internal/
    model/
      model.go                     - shared types (User, Circle, Location, Geofence)
    store/
      store.go                     - DB connection, interfaces, Store struct
      migrations.go                - embedded SQL, auto-migrate on connect
      store_test.go                - integration tests (needs PostGIS)
      migrations/
        001_initial.sql            - schema DDL
    auth/
      auth.go                      - JWT issue/validate, bcrypt hash/check, invite codes
      auth_test.go                 - unit tests
      middleware.go                - HTTP middleware extracting JWT claims
    api/
      server.go                    - chi router, handler registration
      auth_handlers.go             - POST /auth/register, /auth/login
      auth_handlers_test.go        - unit tests with mock store
      location_handlers.go         - POST /locations, GET /locations/latest, /locations/history
      location_handlers_test.go
      circle_handlers.go           - POST /circles, POST /circles/:id/join, GET /circles/:id/members
      circle_handlers_test.go
      geofence_handlers.go         - CRUD /geofences
      geofence_handlers_test.go
    ws/
      hub.go                       - WebSocket hub, connection management, broadcast
      hub_test.go
    geo/
      geo.go                       - geofence state tracker, enter/leave detection
      geo_test.go
    notify/
      notify.go                    - FCM client wrapper
      notify_test.go
  go.mod
  go.sum
docker-compose.yml
Dockerfile
.env.example
```

---

## Task 1: Project Scaffolding + Docker Compose

**Files:**
- Create: `server/go.mod`
- Create: `server/cmd/tracker/main.go`
- Create: `docker-compose.yml`
- Create: `Dockerfile`
- Create: `.env.example`
- Create: `.gitignore`

- [ ] **Step 1: Initialize Go module**

```bash
cd server && go mod init github.com/nschatz/tracker/server
```

- [ ] **Step 2: Create minimal main.go**

Create `server/cmd/tracker/main.go`:

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
```

- [ ] **Step 3: Create docker-compose.yml**

Create `docker-compose.yml`:

```yaml
services:
  postgres:
    image: postgis/postgis:16-3.4
    environment:
      POSTGRES_USER: tracker
      POSTGRES_PASSWORD: tracker
      POSTGRES_DB: tracker
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U tracker"]
      interval: 5s
      timeout: 3s
      retries: 5

  tracker-server:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://tracker:tracker@postgres:5432/tracker?sslmode=disable
      JWT_SECRET: dev-secret-change-me
      PORT: "8080"
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  pgdata:
```

- [ ] **Step 4: Create Dockerfile**

Create `Dockerfile`:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
RUN CGO_ENABLED=0 go build -o /tracker ./cmd/tracker

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /tracker /usr/local/bin/tracker
ENTRYPOINT ["tracker"]
```

- [ ] **Step 5: Create .env.example**

Create `.env.example`:

```
DATABASE_URL=postgres://tracker:tracker@localhost:5432/tracker?sslmode=disable
JWT_SECRET=change-me-in-production
FCM_CREDENTIALS_FILE=
LOCATION_RETENTION_DAYS=30
WS_PING_INTERVAL=30s
PORT=8080
```

- [ ] **Step 6: Create .gitignore**

Create `.gitignore`:

```
.env
*.exe
/server/tracker
.superpowers/
```

- [ ] **Step 7: Verify it builds and runs**

```bash
cd server && go build ./cmd/tracker
docker compose up -d postgres
curl http://localhost:8080/health
# Expected: "ok"
```

- [ ] **Step 8: Commit**

```bash
git init
git add -A
git commit -m "feat: project scaffolding with Docker Compose and health endpoint"
```

---

## Task 2: Database Schema + Migration Runner

**Files:**
- Create: `server/internal/store/migrations/001_initial.sql`
- Create: `server/internal/store/migrations.go`
- Create: `server/internal/store/store.go`

- [ ] **Step 1: Create initial migration SQL**

Create `server/internal/store/migrations/001_initial.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE circles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    invite_code TEXT UNIQUE NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE circle_members (
    circle_id UUID NOT NULL REFERENCES circles(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (circle_id, user_id)
);

CREATE TABLE locations (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    point GEOGRAPHY(Point, 4326) NOT NULL,
    speed REAL,
    battery_level SMALLINT,
    accuracy REAL,
    recorded_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_locations_user_time ON locations (user_id, recorded_at DESC);
CREATE INDEX idx_locations_point ON locations USING GIST (point);
CREATE INDEX idx_locations_recorded_at ON locations USING BRIN (recorded_at);

CREATE TABLE geofences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    circle_id UUID NOT NULL REFERENCES circles(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    center GEOGRAPHY(Point, 4326) NOT NULL,
    radius_meters REAL NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

- [ ] **Step 2: Create migration runner**

Create `server/internal/store/migrations.go`:

```go
package store

import (
	"context"
	"embed"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		version, err := strconv.Atoi(strings.Split(name, "_")[0])
		if err != nil {
			return fmt.Errorf("parse version from %s: %w", name, err)
		}

		var exists bool
		err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if exists {
			continue
		}

		sql, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx for migration %d: %w", version, err)
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("run migration %d: %w", version, err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("record migration %d: %w", version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}

		log.Printf("applied migration %03d: %s", version, name)
	}

	return nil
}
```

- [ ] **Step 3: Create store connection**

Create `server/internal/store/store.go`:

```go
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := RunMigrations(ctx, pool); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}
```

- [ ] **Step 4: Install dependencies**

```bash
cd server && go get github.com/jackc/pgx/v5
```

- [ ] **Step 5: Verify migrations run against PostGIS**

```bash
docker compose up -d postgres
cd server && DATABASE_URL="postgres://tracker:tracker@localhost:5432/tracker?sslmode=disable" go run ./cmd/tracker
# Expected: log line "applied migration 001: 001_initial.sql" then "listening on :8080"
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: database schema with PostGIS and auto-migration runner"
```

---

## Task 3: Shared Model Types

**Files:**
- Create: `server/internal/model/model.go`

- [ ] **Step 1: Define all shared types**

Create `server/internal/model/model.go`:

```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Circle struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	InviteCode string    `json:"invite_code"`
	CreatedBy  uuid.UUID `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type CircleMember struct {
	CircleID    uuid.UUID `json:"circle_id"`
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
}

type Location struct {
	ID           int64     `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	Speed        *float32  `json:"speed,omitempty"`
	BatteryLevel *int16    `json:"battery_level,omitempty"`
	Accuracy     *float32  `json:"accuracy,omitempty"`
	RecordedAt   time.Time `json:"recorded_at"`
}

type LocationInput struct {
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	Speed        *float32  `json:"speed,omitempty"`
	BatteryLevel *int16    `json:"battery_level,omitempty"`
	Accuracy     *float32  `json:"accuracy,omitempty"`
	RecordedAt   time.Time `json:"recorded_at"`
}

type Geofence struct {
	ID           uuid.UUID `json:"id"`
	CircleID     uuid.UUID `json:"circle_id"`
	Name         string    `json:"name"`
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	RadiusMeters float32   `json:"radius_meters"`
	CreatedBy    uuid.UUID `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}
```

- [ ] **Step 2: Install uuid dependency**

```bash
cd server && go get github.com/google/uuid
```

- [ ] **Step 3: Verify it compiles**

```bash
cd server && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: shared model types for users, circles, locations, geofences"
```

---

## Task 4: Store — User + Circle Operations

**Files:**
- Create: `server/internal/store/users.go`
- Create: `server/internal/store/circles.go`
- Create: `server/internal/store/store_test.go`
- Modify: `server/internal/store/store.go` (add interfaces)

- [ ] **Step 1: Write integration tests for user operations**

Create `server/internal/store/store_test.go`:

```go
package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/nschatz/tracker/server/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://tracker:tracker@localhost:5432/tracker?sslmode=disable"
	}
	ctx := context.Background()
	s, err := store.New(ctx, url)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGetUser(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	user, err := s.CreateUser(ctx, "test@example.com", "Test User", "hashedpw")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("email = %q, want test@example.com", user.Email)
	}
	if user.DisplayName != "Test User" {
		t.Errorf("display_name = %q, want Test User", user.DisplayName)
	}

	got, err := s.GetUserByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("id = %v, want %v", got.ID, user.ID)
	}

	got2, err := s.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}
	if got2.Email != "test@example.com" {
		t.Errorf("email = %q, want test@example.com", got2.Email)
	}
}

func TestCreateCircleAndJoin(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	owner, err := s.CreateUser(ctx, "owner@example.com", "Owner", "hashedpw")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}

	circle, err := s.CreateCircle(ctx, "Family", owner.ID)
	if err != nil {
		t.Fatalf("create circle: %v", err)
	}
	if circle.Name != "Family" {
		t.Errorf("name = %q, want Family", circle.Name)
	}
	if circle.InviteCode == "" {
		t.Error("invite_code is empty")
	}

	member, err := s.CreateUser(ctx, "member@example.com", "Member", "hashedpw")
	if err != nil {
		t.Fatalf("create member: %v", err)
	}

	found, err := s.GetCircleByInviteCode(ctx, circle.InviteCode)
	if err != nil {
		t.Fatalf("get circle by invite code: %v", err)
	}
	if found.ID != circle.ID {
		t.Errorf("circle id mismatch")
	}

	err = s.AddMember(ctx, circle.ID, member.ID, "member")
	if err != nil {
		t.Fatalf("add member: %v", err)
	}

	members, err := s.GetMembers(ctx, circle.ID)
	if err != nil {
		t.Fatalf("get members: %v", err)
	}
	// Owner (auto-added as admin) + member
	if len(members) != 2 {
		t.Errorf("len(members) = %d, want 2", len(members))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd server && go test ./internal/store/ -v -count=1
# Expected: FAIL — CreateUser, GetUserByEmail etc. not defined
```

- [ ] **Step 3: Implement user operations**

Create `server/internal/store/users.go`:

```go
package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/model"
)

func (s *Store) CreateUser(ctx context.Context, email, displayName, passwordHash string) (*model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, display_name, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, email, display_name, password_hash, created_at`,
		email, displayName, passwordHash,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, display_name, password_hash, created_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, display_name, password_hash, created_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}
```

- [ ] **Step 4: Implement circle operations**

Create `server/internal/store/circles.go`:

```go
package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/model"
)

func generateInviteCode() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) CreateCircle(ctx context.Context, name string, createdBy uuid.UUID) (*model.Circle, error) {
	code := generateInviteCode()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var c model.Circle
	err = tx.QueryRow(ctx,
		`INSERT INTO circles (name, invite_code, created_by)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, invite_code, created_by, created_at`,
		name, code, createdBy,
	).Scan(&c.ID, &c.Name, &c.InviteCode, &c.CreatedBy, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert circle: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO circle_members (circle_id, user_id, role) VALUES ($1, $2, 'admin')`,
		c.ID, createdBy,
	)
	if err != nil {
		return nil, fmt.Errorf("add creator as admin: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &c, nil
}

func (s *Store) GetCircleByInviteCode(ctx context.Context, code string) (*model.Circle, error) {
	var c model.Circle
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, invite_code, created_by, created_at
		 FROM circles WHERE invite_code = $1`,
		code,
	).Scan(&c.ID, &c.Name, &c.InviteCode, &c.CreatedBy, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get circle by invite code: %w", err)
	}
	return &c, nil
}

func (s *Store) AddMember(ctx context.Context, circleID, userID uuid.UUID, role string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO circle_members (circle_id, user_id, role)
		 VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`,
		circleID, userID, role,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (s *Store) GetMembers(ctx context.Context, circleID uuid.UUID) ([]model.CircleMember, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT cm.circle_id, cm.user_id, cm.role, cm.joined_at, u.display_name, u.email
		 FROM circle_members cm
		 JOIN users u ON u.id = cm.user_id
		 WHERE cm.circle_id = $1
		 ORDER BY cm.joined_at`,
		circleID,
	)
	if err != nil {
		return nil, fmt.Errorf("query members: %w", err)
	}
	defer rows.Close()

	var members []model.CircleMember
	for rows.Next() {
		var m model.CircleMember
		if err := rows.Scan(&m.CircleID, &m.UserID, &m.Role, &m.JoinedAt, &m.DisplayName, &m.Email); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (s *Store) GetUserCircles(ctx context.Context, userID uuid.UUID) ([]model.Circle, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT c.id, c.name, c.invite_code, c.created_by, c.created_at
		 FROM circles c
		 JOIN circle_members cm ON cm.circle_id = c.id
		 WHERE cm.user_id = $1
		 ORDER BY c.created_at`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query user circles: %w", err)
	}
	defer rows.Close()

	var circles []model.Circle
	for rows.Next() {
		var c model.Circle
		if err := rows.Scan(&c.ID, &c.Name, &c.InviteCode, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan circle: %w", err)
		}
		circles = append(circles, c)
	}
	return circles, rows.Err()
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd server && go test ./internal/store/ -v -count=1
# Expected: PASS — TestCreateAndGetUser, TestCreateCircleAndJoin
```

Note: tests require PostGIS running (`docker compose up -d postgres`). Each test run creates new rows with unique emails; for a clean slate, recreate the DB: `docker compose down -v && docker compose up -d postgres`.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: store layer for users and circles with integration tests"
```

---

## Task 5: Auth Package — JWT + bcrypt

**Files:**
- Create: `server/internal/auth/auth.go`
- Create: `server/internal/auth/auth_test.go`
- Create: `server/internal/auth/middleware.go`

- [ ] **Step 1: Write auth unit tests**

Create `server/internal/auth/auth_test.go`:

```go
package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/auth"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := auth.HashPassword("mysecretpassword")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !auth.CheckPassword(hash, "mysecretpassword") {
		t.Error("CheckPassword returned false for correct password")
	}
	if auth.CheckPassword(hash, "wrongpassword") {
		t.Error("CheckPassword returned true for wrong password")
	}
}

func TestIssueAndParseToken(t *testing.T) {
	a := auth.New("test-secret")

	userID := uuid.New()
	token, err := a.IssueToken(userID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	parsed, err := a.ParseToken(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed != userID {
		t.Errorf("parsed = %v, want %v", parsed, userID)
	}
}

func TestParseTokenInvalid(t *testing.T) {
	a := auth.New("test-secret")
	_, err := a.ParseToken("garbage")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestMiddleware(t *testing.T) {
	a := auth.New("test-secret")
	userID := uuid.New()
	token, _ := a.IssueToken(userID)

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := auth.UserIDFromContext(r.Context())
		if got != userID {
			t.Errorf("got = %v, want %v", got, userID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestMiddlewareNoToken(t *testing.T) {
	a := auth.New("test-secret")

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd server && go test ./internal/auth/ -v -count=1
# Expected: FAIL — package auth not found / functions not defined
```

- [ ] **Step 3: Implement auth package**

Create `server/internal/auth/auth.go`:

```go
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	secret []byte
}

func New(secret string) *Auth {
	return &Auth{secret: []byte(secret)}
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (a *Auth) IssueToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(a.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (a *Auth) ParseToken(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.secret, nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("missing sub claim")
	}

	id, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse sub as uuid: %w", err)
	}
	return id, nil
}
```

- [ ] **Step 4: Implement auth middleware**

Create `server/internal/auth/middleware.go`:

```go
package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey struct{}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		userID, err := a.ParseToken(tokenStr)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(contextKey{}).(uuid.UUID)
	return id
}
```

- [ ] **Step 5: Install dependencies and run tests**

```bash
cd server && go get github.com/golang-jwt/jwt/v5 golang.org/x/crypto
cd server && go test ./internal/auth/ -v -count=1
# Expected: PASS — all 5 tests
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: auth package with JWT, bcrypt, and HTTP middleware"
```

---

## Task 6: API — Router Scaffold + Auth Endpoints

**Files:**
- Create: `server/internal/api/server.go`
- Create: `server/internal/api/auth_handlers.go`
- Create: `server/internal/api/auth_handlers_test.go`

- [ ] **Step 1: Write auth handler tests**

Create `server/internal/api/auth_handlers_test.go`:

```go
package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/api"
	"github.com/nschatz/tracker/server/internal/auth"
	"github.com/nschatz/tracker/server/internal/model"
)

type mockStore struct {
	users   map[string]*model.User
	circles map[string]*model.Circle
}

func newMockStore() *mockStore {
	return &mockStore{
		users:   make(map[string]*model.User),
		circles: make(map[string]*model.Circle),
	}
}

func (m *mockStore) CreateUser(ctx context.Context, email, displayName, passwordHash string) (*model.User, error) {
	u := &model.User{
		ID:           uuid.New(),
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
	}
	m.users[email] = u
	return u, nil
}

func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return u, nil
}

func (m *mockStore) GetCircleByInviteCode(ctx context.Context, code string) (*model.Circle, error) {
	c, ok := m.circles[code]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return c, nil
}

func (m *mockStore) AddMember(ctx context.Context, circleID, userID uuid.UUID, role string) error {
	return nil
}

func TestRegisterAndLogin(t *testing.T) {
	a := auth.New("test-secret")
	ms := newMockStore()

	// Pre-create a circle with invite code for registration
	circleID := uuid.New()
	ms.circles["abc123"] = &model.Circle{ID: circleID, InviteCode: "abc123"}

	srv := api.NewServer(a, ms)

	// Register
	body, _ := json.Marshal(map[string]string{
		"email":        "test@example.com",
		"display_name": "Test",
		"password":     "secret123",
		"invite_code":  "abc123",
	})
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201, body: %s", rec.Code, rec.Body.String())
	}

	var regResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &regResp)
	if regResp["token"] == nil {
		t.Error("register response missing token")
	}

	// Login
	body, _ = json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "secret123",
	})
	req = httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var loginResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &loginResp)
	if loginResp["token"] == nil {
		t.Error("login response missing token")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd server && go test ./internal/api/ -v -count=1
# Expected: FAIL — api package not found
```

- [ ] **Step 3: Define store interfaces in server.go**

Create `server/internal/api/server.go`:

```go
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
```

- [ ] **Step 4: Implement auth handlers**

Create `server/internal/api/auth_handlers.go`:

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/nschatz/tracker/server/internal/auth"
)

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	InviteCode  string `json:"invite_code"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	} `json:"user"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.InviteCode == "" || req.DisplayName == "" {
		http.Error(w, "email, display_name, password, and invite_code are required", http.StatusBadRequest)
		return
	}

	circle, err := s.store.GetCircleByInviteCode(r.Context(), req.InviteCode)
	if err != nil {
		http.Error(w, "invalid invite code", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	user, err := s.store.CreateUser(r.Context(), req.Email, req.DisplayName, hash)
	if err != nil {
		http.Error(w, "could not create user (email may already exist)", http.StatusConflict)
		return
	}

	if err := s.store.AddMember(r.Context(), circle.ID, user.ID, "member"); err != nil {
		http.Error(w, "could not join circle", http.StatusInternalServerError)
		return
	}

	token, err := s.auth.IssueToken(user.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := authResponse{Token: token}
	resp.User.ID = user.ID.String()
	resp.User.Email = user.Email
	resp.User.DisplayName = user.DisplayName

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := s.auth.IssueToken(user.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := authResponse{Token: token}
	resp.User.ID = user.ID.String()
	resp.User.Email = user.Email
	resp.User.DisplayName = user.DisplayName

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
```

- [ ] **Step 5: Install chi and fix test imports, then run tests**

```bash
cd server && go get github.com/go-chi/chi/v5
cd server && go test ./internal/api/ -v -count=1
# Expected: PASS — TestRegisterAndLogin
```

Note: the test file needs `"fmt"` imported for the mock store. Add it if the compiler complains.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: API router with register and login endpoints"
```

---

## Task 7: Store — Location Operations

**Files:**
- Create: `server/internal/store/locations.go`
- Modify: `server/internal/store/store_test.go` (add location tests)

- [ ] **Step 1: Write location store tests**

Append to `server/internal/store/store_test.go`:

```go
func TestInsertAndQueryLocations(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	user, _ := s.CreateUser(ctx, "locuser@example.com", "Loc User", "hash")
	owner, _ := s.CreateUser(ctx, "locowner@example.com", "Loc Owner", "hash")
	circle, _ := s.CreateCircle(ctx, "LocCircle", owner.ID)
	s.AddMember(ctx, circle.ID, user.ID, "member")

	now := time.Now()
	inputs := []model.LocationInput{
		{Lat: 40.7128, Lng: -74.0060, RecordedAt: now.Add(-2 * time.Minute)},
		{Lat: 40.7130, Lng: -74.0062, RecordedAt: now.Add(-1 * time.Minute)},
		{Lat: 40.7135, Lng: -74.0065, RecordedAt: now},
	}

	err := s.InsertLocations(ctx, user.ID, inputs)
	if err != nil {
		t.Fatalf("insert locations: %v", err)
	}

	// Test latest locations
	latest, err := s.GetLatestLocations(ctx, circle.ID)
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if len(latest) < 1 {
		t.Fatal("expected at least 1 latest location")
	}
	found := false
	for _, loc := range latest {
		if loc.UserID == user.ID {
			found = true
			if loc.Lat != 40.7135 {
				t.Errorf("latest lat = %f, want 40.7135", loc.Lat)
			}
		}
	}
	if !found {
		t.Error("user not found in latest locations")
	}

	// Test history
	history, err := s.GetHistory(ctx, user.ID, now.Add(-3*time.Minute), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("history len = %d, want 3", len(history))
	}
}
```

Add `"time"` and `"github.com/nschatz/tracker/server/internal/model"` to the test file imports.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd server && go test ./internal/store/ -run TestInsertAndQueryLocations -v -count=1
# Expected: FAIL — InsertLocations not defined
```

- [ ] **Step 3: Implement location store operations**

Create `server/internal/store/locations.go`:

```go
package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/model"
)

func (s *Store) InsertLocations(ctx context.Context, userID uuid.UUID, locs []model.LocationInput) error {
	if len(locs) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("INSERT INTO locations (user_id, point, speed, battery_level, accuracy, recorded_at) VALUES ")

	args := make([]interface{}, 0, len(locs)*6)
	for i, loc := range locs {
		if i > 0 {
			b.WriteString(", ")
		}
		base := i * 6
		fmt.Fprintf(&b, "($%d, ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6)
		args = append(args, userID, loc.Lng, loc.Lat, loc.Speed, loc.BatteryLevel, loc.Accuracy)
		// Note: ST_MakePoint takes (lng, lat) order
	}

	// Append recorded_at — we need 7 params per row, not 6
	// Let me redo this properly

	var b2 strings.Builder
	b2.WriteString("INSERT INTO locations (user_id, point, speed, battery_level, accuracy, recorded_at) VALUES ")

	args = args[:0]
	for i, loc := range locs {
		if i > 0 {
			b2.WriteString(", ")
		}
		base := i*7 + 1
		fmt.Fprintf(&b2, "($%d, ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography, $%d, $%d, $%d, $%d)",
			base, base+1, base+2, base+3, base+4, base+5, base+6)
		args = append(args, userID, loc.Lng, loc.Lat, loc.Speed, loc.BatteryLevel, loc.Accuracy, loc.RecordedAt)
	}

	_, err := s.pool.Exec(ctx, b2.String(), args...)
	if err != nil {
		return fmt.Errorf("insert locations: %w", err)
	}
	return nil
}

func (s *Store) GetLatestLocations(ctx context.Context, circleID uuid.UUID) ([]model.Location, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (l.user_id)
			l.id, l.user_id,
			ST_Y(l.point::geometry) AS lat,
			ST_X(l.point::geometry) AS lng,
			l.speed, l.battery_level, l.accuracy, l.recorded_at
		FROM locations l
		JOIN circle_members cm ON cm.user_id = l.user_id
		WHERE cm.circle_id = $1
		ORDER BY l.user_id, l.recorded_at DESC
	`, circleID)
	if err != nil {
		return nil, fmt.Errorf("query latest locations: %w", err)
	}
	defer rows.Close()

	var locs []model.Location
	for rows.Next() {
		var loc model.Location
		if err := rows.Scan(&loc.ID, &loc.UserID, &loc.Lat, &loc.Lng, &loc.Speed, &loc.BatteryLevel, &loc.Accuracy, &loc.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan location: %w", err)
		}
		locs = append(locs, loc)
	}
	return locs, rows.Err()
}

func (s *Store) GetHistory(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]model.Location, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id,
			ST_Y(point::geometry) AS lat,
			ST_X(point::geometry) AS lng,
			speed, battery_level, accuracy, recorded_at
		FROM locations
		WHERE user_id = $1 AND recorded_at >= $2 AND recorded_at <= $3
		ORDER BY recorded_at ASC
	`, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var locs []model.Location
	for rows.Next() {
		var loc model.Location
		if err := rows.Scan(&loc.ID, &loc.UserID, &loc.Lat, &loc.Lng, &loc.Speed, &loc.BatteryLevel, &loc.Accuracy, &loc.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan location: %w", err)
		}
		locs = append(locs, loc)
	}
	return locs, rows.Err()
}

func (s *Store) DeleteLocationsOlderThan(ctx context.Context, days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	tag, err := s.pool.Exec(ctx, "DELETE FROM locations WHERE recorded_at < $1", cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old locations: %w", err)
	}
	return tag.RowsAffected(), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd server && go test ./internal/store/ -run TestInsertAndQueryLocations -v -count=1
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: store layer for location insert, latest, history, and retention"
```

---

## Task 8: API — Location Endpoints

**Files:**
- Create: `server/internal/api/location_handlers.go`
- Create: `server/internal/api/location_handlers_test.go`
- Modify: `server/internal/api/server.go` (add LocationStore interface, wire routes)

- [ ] **Step 1: Add LocationStore interface to server.go**

Add to `server/internal/api/server.go`:

```go
type LocationStore interface {
	InsertLocations(ctx context.Context, userID uuid.UUID, locs []model.LocationInput) error
	GetLatestLocations(ctx context.Context, circleID uuid.UUID) ([]model.Location, error)
	GetHistory(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]model.Location, error)
}
```

Update the `Server` struct to include `locations LocationStore` and update `NewServer` to accept it. Add authenticated routes:

```go
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
```

Update the `time` import.

- [ ] **Step 2: Write location handler tests**

Create `server/internal/api/location_handlers_test.go`:

```go
package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/api"
	"github.com/nschatz/tracker/server/internal/auth"
	"github.com/nschatz/tracker/server/internal/model"
)

type mockLocationStore struct {
	locations []model.Location
}

func (m *mockLocationStore) InsertLocations(ctx context.Context, userID uuid.UUID, locs []model.LocationInput) error {
	for _, l := range locs {
		m.locations = append(m.locations, model.Location{
			UserID: userID, Lat: l.Lat, Lng: l.Lng, RecordedAt: l.RecordedAt,
		})
	}
	return nil
}

func (m *mockLocationStore) GetLatestLocations(ctx context.Context, circleID uuid.UUID) ([]model.Location, error) {
	return m.locations, nil
}

func (m *mockLocationStore) GetHistory(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]model.Location, error) {
	return m.locations, nil
}

func TestPostLocations(t *testing.T) {
	a := auth.New("test-secret")
	ms := newMockStore()
	ls := &mockLocationStore{}
	srv := api.NewServer(a, ms, ls)

	userID := uuid.New()
	token, _ := a.IssueToken(userID)

	body, _ := json.Marshal(map[string]interface{}{
		"locations": []map[string]interface{}{
			{"lat": 40.7128, "lng": -74.0060, "recorded_at": time.Now().Format(time.RFC3339)},
		},
	})

	req := httptest.NewRequest("POST", "/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202, body: %s", rec.Code, rec.Body.String())
	}
	if len(ls.locations) != 1 {
		t.Errorf("stored %d locations, want 1", len(ls.locations))
	}
}
```

- [ ] **Step 3: Implement location handlers**

Create `server/internal/api/location_handlers.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/auth"
	"github.com/nschatz/tracker/server/internal/model"
)

type postLocationsRequest struct {
	Locations []model.LocationInput `json:"locations"`
}

func (s *Server) handlePostLocations(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req postLocationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Locations) == 0 {
		http.Error(w, "at least one location required", http.StatusBadRequest)
		return
	}

	if err := s.locations.InsertLocations(r.Context(), userID, req.Locations); err != nil {
		http.Error(w, "failed to store locations", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleGetLatestLocations(w http.ResponseWriter, r *http.Request) {
	circleIDStr := r.URL.Query().Get("circle_id")
	if circleIDStr == "" {
		http.Error(w, "circle_id query param required", http.StatusBadRequest)
		return
	}

	circleID, err := uuid.Parse(circleIDStr)
	if err != nil {
		http.Error(w, "invalid circle_id", http.StatusBadRequest)
		return
	}

	locs, err := s.locations.GetLatestLocations(r.Context(), circleID)
	if err != nil {
		http.Error(w, "failed to get locations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(locs)
}

func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if userIDStr == "" || fromStr == "" || toStr == "" {
		http.Error(w, "user_id, from, and to query params required", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		http.Error(w, "invalid from (use RFC3339)", http.StatusBadRequest)
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		http.Error(w, "invalid to (use RFC3339)", http.StatusBadRequest)
		return
	}

	locs, err := s.locations.GetHistory(r.Context(), userID, from, to)
	if err != nil {
		http.Error(w, "failed to get history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(locs)
}
```

- [ ] **Step 4: Fix auth_handlers_test.go to pass updated NewServer signature**

Update the `TestRegisterAndLogin` test to pass a `nil` or mock location store:

```go
srv := api.NewServer(a, ms, &mockLocationStore{})
```

- [ ] **Step 5: Run all API tests**

```bash
cd server && go test ./internal/api/ -v -count=1
# Expected: PASS — TestRegisterAndLogin, TestPostLocations
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: location ingestion and query API endpoints"
```

---

## Task 9: WebSocket Hub

**Files:**
- Create: `server/internal/ws/hub.go`
- Create: `server/internal/ws/hub_test.go`

- [ ] **Step 1: Write hub tests**

Create `server/internal/ws/hub_test.go`:

```go
package ws_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/model"
	"github.com/nschatz/tracker/server/internal/ws"

	"nhooyr.io/websocket"
)

func TestHubBroadcast(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	circleID := uuid.New()
	userA := uuid.New()
	userB := uuid.New()

	// Start WebSocket test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleConnect(w, r, userB, circleID)
	}))
	defer server.Close()

	// Connect as userB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + server.URL[4:] // http -> ws
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Give the connection time to register
	time.Sleep(100 * time.Millisecond)

	// Broadcast location from userA in the same circle
	loc := model.Location{
		UserID: userA, Lat: 40.7128, Lng: -74.0060, RecordedAt: time.Now(),
	}
	hub.BroadcastLocation(circleID, loc)

	// Read message as userB
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var got model.Location
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.UserID != userA {
		t.Errorf("got user_id = %v, want %v", got.UserID, userA)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd server && go get nhooyr.io/websocket
cd server && go test ./internal/ws/ -v -count=1
# Expected: FAIL — ws package not found
```

- [ ] **Step 3: Implement WebSocket hub**

Create `server/internal/ws/hub.go`:

```go
package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/model"

	"nhooyr.io/websocket"
)

type client struct {
	conn     *websocket.Conn
	userID   uuid.UUID
	circleID uuid.UUID
	cancel   context.CancelFunc
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*client]struct{}
	register   chan *client
	unregister chan *client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*client]struct{}),
		register:   make(chan *client),
		unregister: make(chan *client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			delete(h.clients, c)
			h.mu.Unlock()
		}
	}
}

func (h *Hub) HandleConnect(w http.ResponseWriter, r *http.Request, userID, circleID uuid.UUID) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("ws accept: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	c := &client{
		conn:     conn,
		userID:   userID,
		circleID: circleID,
		cancel:   cancel,
	}

	h.register <- c

	defer func() {
		h.unregister <- c
		conn.Close(websocket.StatusNormalClosure, "")
		cancel()
	}()

	// Read loop — keeps connection alive, handles client close
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			return
		}
	}
}

func (h *Hub) BroadcastLocation(circleID uuid.UUID, loc model.Location) {
	data, err := json.Marshal(loc)
	if err != nil {
		log.Printf("marshal location: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c.circleID == circleID {
			ctx, cancel := context.WithTimeout(context.Background(), 5*1e9) // 5s
			err := c.conn.Write(ctx, websocket.MessageText, data)
			cancel()
			if err != nil {
				log.Printf("ws write to %v: %v", c.userID, err)
			}
		}
	}
}

// IsConnected returns true if the given user has an active WebSocket connection.
func (h *Hub) IsConnected(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c.userID == userID {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd server && go test ./internal/ws/ -v -count=1
# Expected: PASS — TestHubBroadcast
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: WebSocket hub for real-time location broadcast"
```

---

## Task 10: API — Circle Endpoints

**Files:**
- Create: `server/internal/api/circle_handlers.go`
- Create: `server/internal/api/circle_handlers_test.go`
- Modify: `server/internal/api/server.go` (add CircleStore interface, wire routes)

- [ ] **Step 1: Add CircleStore interface to server.go**

Add to `server/internal/api/server.go`:

```go
type CircleStore interface {
	CreateCircle(ctx context.Context, name string, createdBy uuid.UUID) (*model.Circle, error)
	GetUserCircles(ctx context.Context, userID uuid.UUID) ([]model.Circle, error)
	GetMembers(ctx context.Context, circleID uuid.UUID) ([]model.CircleMember, error)
	GetCircleByInviteCode(ctx context.Context, code string) (*model.Circle, error)
	AddMember(ctx context.Context, circleID, userID uuid.UUID, role string) error
}
```

Update `Server` struct to include `circles CircleStore`. Update `NewServer` to accept it. Move `GetCircleByInviteCode` and `AddMember` out of `AuthStore` and into `CircleStore` (auth handlers will use `circles` field instead). Add routes:

```go
r.Post("/circles", s.handleCreateCircle)
r.Post("/circles/{id}/join", s.handleJoinCircle)
r.Get("/circles/{id}/members", s.handleGetMembers)
r.Get("/circles", s.handleGetUserCircles)
```

- [ ] **Step 2: Write circle handler tests**

Create `server/internal/api/circle_handlers_test.go`:

```go
package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/api"
	"github.com/nschatz/tracker/server/internal/auth"
	"github.com/nschatz/tracker/server/internal/model"
)

type mockCircleStore struct {
	circles map[uuid.UUID]*model.Circle
	members map[uuid.UUID][]model.CircleMember
}

func newMockCircleStore() *mockCircleStore {
	return &mockCircleStore{
		circles: make(map[uuid.UUID]*model.Circle),
		members: make(map[uuid.UUID][]model.CircleMember),
	}
}

func (m *mockCircleStore) CreateCircle(ctx context.Context, name string, createdBy uuid.UUID) (*model.Circle, error) {
	c := &model.Circle{ID: uuid.New(), Name: name, InviteCode: "test-code", CreatedBy: createdBy}
	m.circles[c.ID] = c
	return c, nil
}

func (m *mockCircleStore) GetUserCircles(ctx context.Context, userID uuid.UUID) ([]model.Circle, error) {
	var result []model.Circle
	for _, c := range m.circles {
		result = append(result, *c)
	}
	return result, nil
}

func (m *mockCircleStore) GetMembers(ctx context.Context, circleID uuid.UUID) ([]model.CircleMember, error) {
	return m.members[circleID], nil
}

func (m *mockCircleStore) GetCircleByInviteCode(ctx context.Context, code string) (*model.Circle, error) {
	for _, c := range m.circles {
		if c.InviteCode == code {
			return c, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockCircleStore) AddMember(ctx context.Context, circleID, userID uuid.UUID, role string) error {
	return nil
}

func TestCreateCircle(t *testing.T) {
	a := auth.New("test-secret")
	cs := newMockCircleStore()
	srv := api.NewServer(a, newMockStore(), &mockLocationStore{}, cs)

	userID := uuid.New()
	token, _ := a.IssueToken(userID)

	body, _ := json.Marshal(map[string]string{"name": "Family"})
	req := httptest.NewRequest("POST", "/circles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body: %s", rec.Code, rec.Body.String())
	}

	var resp model.Circle
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Name != "Family" {
		t.Errorf("name = %q, want Family", resp.Name)
	}
}
```

- [ ] **Step 3: Implement circle handlers**

Create `server/internal/api/circle_handlers.go`:

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/auth"
)

type createCircleRequest struct {
	Name string `json:"name"`
}

type joinCircleRequest struct {
	InviteCode string `json:"invite_code"`
}

func (s *Server) handleCreateCircle(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req createCircleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	circle, err := s.circles.CreateCircle(r.Context(), req.Name, userID)
	if err != nil {
		http.Error(w, "failed to create circle", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(circle)
}

func (s *Server) handleJoinCircle(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	circleIDStr := chi.URLParam(r, "id")

	circleID, err := uuid.Parse(circleIDStr)
	if err != nil {
		http.Error(w, "invalid circle id", http.StatusBadRequest)
		return
	}

	var req joinCircleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	circle, err := s.circles.GetCircleByInviteCode(r.Context(), req.InviteCode)
	if err != nil || circle.ID != circleID {
		http.Error(w, "invalid invite code", http.StatusBadRequest)
		return
	}

	if err := s.circles.AddMember(r.Context(), circleID, userID, "member"); err != nil {
		http.Error(w, "failed to join circle", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "joined"})
}

func (s *Server) handleGetMembers(w http.ResponseWriter, r *http.Request) {
	circleIDStr := chi.URLParam(r, "id")

	circleID, err := uuid.Parse(circleIDStr)
	if err != nil {
		http.Error(w, "invalid circle id", http.StatusBadRequest)
		return
	}

	members, err := s.circles.GetMembers(r.Context(), circleID)
	if err != nil {
		http.Error(w, "failed to get members", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

func (s *Server) handleGetUserCircles(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	circles, err := s.circles.GetUserCircles(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to get circles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(circles)
}
```

- [ ] **Step 4: Run all API tests**

```bash
cd server && go test ./internal/api/ -v -count=1
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: circle CRUD API endpoints"
```

---

## Task 11: Store — Geofence Operations

**Files:**
- Create: `server/internal/store/geofences.go`
- Modify: `server/internal/store/store_test.go` (add geofence tests)

- [ ] **Step 1: Write geofence store tests**

Append to `server/internal/store/store_test.go`:

```go
func TestGeofenceCRUD(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	owner, _ := s.CreateUser(ctx, "geoowner@example.com", "Geo Owner", "hash")
	circle, _ := s.CreateCircle(ctx, "GeoCircle", owner.ID)

	// Create
	gf, err := s.CreateGeofence(ctx, circle.ID, "Home", 40.7128, -74.0060, 100, owner.ID)
	if err != nil {
		t.Fatalf("create geofence: %v", err)
	}
	if gf.Name != "Home" {
		t.Errorf("name = %q, want Home", gf.Name)
	}

	// List
	gfs, err := s.GetGeofences(ctx, circle.ID)
	if err != nil {
		t.Fatalf("get geofences: %v", err)
	}
	if len(gfs) != 1 {
		t.Fatalf("len = %d, want 1", len(gfs))
	}

	// Update
	updated, err := s.UpdateGeofence(ctx, gf.ID, "Work", 40.7580, -73.9855, 200)
	if err != nil {
		t.Fatalf("update geofence: %v", err)
	}
	if updated.Name != "Work" {
		t.Errorf("name = %q, want Work", updated.Name)
	}

	// FindContaining — point inside the geofence
	ids, err := s.FindContainingGeofences(ctx, circle.ID, 40.7580, -73.9855)
	if err != nil {
		t.Fatalf("find containing: %v", err)
	}
	if len(ids) != 1 || ids[0] != gf.ID {
		t.Errorf("expected geofence %v, got %v", gf.ID, ids)
	}

	// FindContaining — point outside
	ids2, err := s.FindContainingGeofences(ctx, circle.ID, 0, 0)
	if err != nil {
		t.Fatalf("find containing outside: %v", err)
	}
	if len(ids2) != 0 {
		t.Errorf("expected 0, got %d", len(ids2))
	}

	// Delete
	err = s.DeleteGeofence(ctx, gf.ID)
	if err != nil {
		t.Fatalf("delete geofence: %v", err)
	}
	gfs2, _ := s.GetGeofences(ctx, circle.ID)
	if len(gfs2) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(gfs2))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd server && go test ./internal/store/ -run TestGeofenceCRUD -v -count=1
# Expected: FAIL — CreateGeofence not defined
```

- [ ] **Step 3: Implement geofence store operations**

Create `server/internal/store/geofences.go`:

```go
package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/model"
)

func (s *Store) CreateGeofence(ctx context.Context, circleID uuid.UUID, name string, lat, lng float64, radiusMeters float32, createdBy uuid.UUID) (*model.Geofence, error) {
	var gf model.Geofence
	err := s.pool.QueryRow(ctx, `
		INSERT INTO geofences (circle_id, name, center, radius_meters, created_by)
		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5, $6)
		RETURNING id, circle_id, name,
			ST_Y(center::geometry) AS lat, ST_X(center::geometry) AS lng,
			radius_meters, created_by, created_at
	`, circleID, name, lng, lat, radiusMeters, createdBy,
	).Scan(&gf.ID, &gf.CircleID, &gf.Name, &gf.Lat, &gf.Lng, &gf.RadiusMeters, &gf.CreatedBy, &gf.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert geofence: %w", err)
	}
	return &gf, nil
}

func (s *Store) GetGeofences(ctx context.Context, circleID uuid.UUID) ([]model.Geofence, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, circle_id, name,
			ST_Y(center::geometry) AS lat, ST_X(center::geometry) AS lng,
			radius_meters, created_by, created_at
		FROM geofences
		WHERE circle_id = $1
		ORDER BY created_at
	`, circleID)
	if err != nil {
		return nil, fmt.Errorf("query geofences: %w", err)
	}
	defer rows.Close()

	var gfs []model.Geofence
	for rows.Next() {
		var gf model.Geofence
		if err := rows.Scan(&gf.ID, &gf.CircleID, &gf.Name, &gf.Lat, &gf.Lng, &gf.RadiusMeters, &gf.CreatedBy, &gf.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan geofence: %w", err)
		}
		gfs = append(gfs, gf)
	}
	return gfs, rows.Err()
}

func (s *Store) UpdateGeofence(ctx context.Context, id uuid.UUID, name string, lat, lng float64, radiusMeters float32) (*model.Geofence, error) {
	var gf model.Geofence
	err := s.pool.QueryRow(ctx, `
		UPDATE geofences
		SET name = $2, center = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, radius_meters = $5
		WHERE id = $1
		RETURNING id, circle_id, name,
			ST_Y(center::geometry) AS lat, ST_X(center::geometry) AS lng,
			radius_meters, created_by, created_at
	`, id, name, lng, lat, radiusMeters,
	).Scan(&gf.ID, &gf.CircleID, &gf.Name, &gf.Lat, &gf.Lng, &gf.RadiusMeters, &gf.CreatedBy, &gf.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("update geofence: %w", err)
	}
	return &gf, nil
}

func (s *Store) DeleteGeofence(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM geofences WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete geofence: %w", err)
	}
	return nil
}

func (s *Store) FindContainingGeofences(ctx context.Context, circleID uuid.UUID, lat, lng float64) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM geofences
		WHERE circle_id = $1
		AND ST_DWithin(center, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, radius_meters)
	`, circleID, lng, lat)
	if err != nil {
		return nil, fmt.Errorf("find containing geofences: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan geofence id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd server && go test ./internal/store/ -run TestGeofenceCRUD -v -count=1
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: geofence store with PostGIS spatial queries"
```

---

## Task 12: API — Geofence CRUD Endpoints

**Files:**
- Create: `server/internal/api/geofence_handlers.go`
- Create: `server/internal/api/geofence_handlers_test.go`
- Modify: `server/internal/api/server.go` (add GeofenceStore interface, wire routes)

- [ ] **Step 1: Add GeofenceStore interface to server.go**

Add to `server/internal/api/server.go`:

```go
type GeofenceStore interface {
	CreateGeofence(ctx context.Context, circleID uuid.UUID, name string, lat, lng float64, radiusMeters float32, createdBy uuid.UUID) (*model.Geofence, error)
	GetGeofences(ctx context.Context, circleID uuid.UUID) ([]model.Geofence, error)
	UpdateGeofence(ctx context.Context, id uuid.UUID, name string, lat, lng float64, radiusMeters float32) (*model.Geofence, error)
	DeleteGeofence(ctx context.Context, id uuid.UUID) error
}
```

Update `Server` struct to include `geofences GeofenceStore`. Update `NewServer`. Add routes:

```go
r.Post("/geofences", s.handleCreateGeofence)
r.Get("/geofences", s.handleGetGeofences)
r.Put("/geofences/{id}", s.handleUpdateGeofence)
r.Delete("/geofences/{id}", s.handleDeleteGeofence)
```

- [ ] **Step 2: Implement geofence handlers**

Create `server/internal/api/geofence_handlers.go`:

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/auth"
)

type createGeofenceRequest struct {
	CircleID     uuid.UUID `json:"circle_id"`
	Name         string    `json:"name"`
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	RadiusMeters float32   `json:"radius_meters"`
}

type updateGeofenceRequest struct {
	Name         string  `json:"name"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	RadiusMeters float32 `json:"radius_meters"`
}

func (s *Server) handleCreateGeofence(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req createGeofenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.RadiusMeters <= 0 {
		http.Error(w, "name and positive radius_meters are required", http.StatusBadRequest)
		return
	}

	gf, err := s.geofences.CreateGeofence(r.Context(), req.CircleID, req.Name, req.Lat, req.Lng, req.RadiusMeters, userID)
	if err != nil {
		http.Error(w, "failed to create geofence", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(gf)
}

func (s *Server) handleGetGeofences(w http.ResponseWriter, r *http.Request) {
	circleIDStr := r.URL.Query().Get("circle_id")
	if circleIDStr == "" {
		http.Error(w, "circle_id query param required", http.StatusBadRequest)
		return
	}

	circleID, err := uuid.Parse(circleIDStr)
	if err != nil {
		http.Error(w, "invalid circle_id", http.StatusBadRequest)
		return
	}

	gfs, err := s.geofences.GetGeofences(r.Context(), circleID)
	if err != nil {
		http.Error(w, "failed to get geofences", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gfs)
}

func (s *Server) handleUpdateGeofence(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid geofence id", http.StatusBadRequest)
		return
	}

	var req updateGeofenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	gf, err := s.geofences.UpdateGeofence(r.Context(), id, req.Name, req.Lat, req.Lng, req.RadiusMeters)
	if err != nil {
		http.Error(w, "failed to update geofence", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gf)
}

func (s *Server) handleDeleteGeofence(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid geofence id", http.StatusBadRequest)
		return
	}

	if err := s.geofences.DeleteGeofence(r.Context(), id); err != nil {
		http.Error(w, "failed to delete geofence", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 3: Write and run geofence handler tests**

Create `server/internal/api/geofence_handlers_test.go` with a mock `GeofenceStore` and test `TestCreateGeofence`. Follow the same mock pattern as previous handler tests.

- [ ] **Step 4: Run all API tests**

```bash
cd server && go test ./internal/api/ -v -count=1
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: geofence CRUD API endpoints"
```

---

## Task 13: Geofence Evaluation (geo package)

**Files:**
- Create: `server/internal/geo/geo.go`
- Create: `server/internal/geo/geo_test.go`

- [ ] **Step 1: Write geofence evaluator tests**

Create `server/internal/geo/geo_test.go`:

```go
package geo_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/geo"
)

func TestDetectTransitions(t *testing.T) {
	tracker := geo.NewTracker()

	userID := uuid.New()
	gfHome := uuid.New()
	gfWork := uuid.New()

	// Initially user is nowhere
	entered, left := tracker.Update(userID, []uuid.UUID{gfHome})
	if len(entered) != 1 || entered[0] != gfHome {
		t.Errorf("entered = %v, want [%v]", entered, gfHome)
	}
	if len(left) != 0 {
		t.Errorf("left = %v, want []", left)
	}

	// User moves from home to work
	entered2, left2 := tracker.Update(userID, []uuid.UUID{gfWork})
	if len(entered2) != 1 || entered2[0] != gfWork {
		t.Errorf("entered = %v, want [%v]", entered2, gfWork)
	}
	if len(left2) != 1 || left2[0] != gfHome {
		t.Errorf("left = %v, want [%v]", left2, gfHome)
	}

	// User stays at work — no transitions
	entered3, left3 := tracker.Update(userID, []uuid.UUID{gfWork})
	if len(entered3) != 0 {
		t.Errorf("entered = %v, want []", entered3)
	}
	if len(left3) != 0 {
		t.Errorf("left = %v, want []", left3)
	}

	// User leaves work, goes nowhere
	entered4, left4 := tracker.Update(userID, []uuid.UUID{})
	if len(entered4) != 0 {
		t.Errorf("entered = %v, want []", entered4)
	}
	if len(left4) != 1 || left4[0] != gfWork {
		t.Errorf("left = %v, want [%v]", left4, gfWork)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd server && go test ./internal/geo/ -v -count=1
# Expected: FAIL — geo package not found
```

- [ ] **Step 3: Implement geofence state tracker**

Create `server/internal/geo/geo.go`:

```go
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

	prev := t.state[userID]
	if prev == nil {
		prev = make(map[uuid.UUID]struct{})
	}

	curr := make(map[uuid.UUID]struct{}, len(currentGeofences))
	for _, id := range currentGeofences {
		curr[id] = struct{}{}
	}

	// Entered: in curr but not in prev
	for id := range curr {
		if _, ok := prev[id]; !ok {
			entered = append(entered, id)
		}
	}

	// Left: in prev but not in curr
	for id := range prev {
		if _, ok := curr[id]; !ok {
			left = append(left, id)
		}
	}

	t.state[userID] = curr
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd server && go test ./internal/geo/ -v -count=1
# Expected: PASS — TestDetectTransitions
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: in-memory geofence state tracker with enter/leave detection"
```

---

## Task 14: FCM Notifications (notify package)

**Files:**
- Create: `server/internal/notify/notify.go`
- Create: `server/internal/notify/notify_test.go`

- [ ] **Step 1: Write notify tests with a mock sender**

Create `server/internal/notify/notify_test.go`:

```go
package notify_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/notify"
)

type mockSender struct {
	sent []notify.Message
}

func (m *mockSender) Send(ctx context.Context, msg notify.Message) error {
	m.sent = append(m.sent, msg)
	return nil
}

func TestNotifyGeofenceEnter(t *testing.T) {
	ms := &mockSender{}
	n := notify.NewNotifier(ms)

	ctx := context.Background()
	n.GeofenceEnter(ctx, "Alice", "Home", []string{"token-bob"})

	if len(ms.sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(ms.sent))
	}
	if ms.sent[0].Title != "Place Alert" {
		t.Errorf("title = %q, want Place Alert", ms.sent[0].Title)
	}
}

func TestNotifyGeofenceLeave(t *testing.T) {
	ms := &mockSender{}
	n := notify.NewNotifier(ms)

	ctx := context.Background()
	n.GeofenceLeave(ctx, "Alice", "Work", []string{"token-bob", "token-charlie"})

	if len(ms.sent) != 2 {
		t.Fatalf("sent %d messages, want 2", len(ms.sent))
	}
}

// Ensure unused import doesn't cause issues
var _ = uuid.New
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd server && go test ./internal/notify/ -v -count=1
# Expected: FAIL — notify package not found
```

- [ ] **Step 3: Implement notify package**

Create `server/internal/notify/notify.go`:

```go
package notify

import (
	"context"
	"fmt"
	"log"
)

type Message struct {
	Token string
	Title string
	Body  string
}

type Sender interface {
	Send(ctx context.Context, msg Message) error
}

type Notifier struct {
	sender Sender
}

func NewNotifier(sender Sender) *Notifier {
	return &Notifier{sender: sender}
}

func (n *Notifier) GeofenceEnter(ctx context.Context, userName, placeName string, fcmTokens []string) {
	body := fmt.Sprintf("%s arrived at %s", userName, placeName)
	n.sendToAll(ctx, "Place Alert", body, fcmTokens)
}

func (n *Notifier) GeofenceLeave(ctx context.Context, userName, placeName string, fcmTokens []string) {
	body := fmt.Sprintf("%s left %s", userName, placeName)
	n.sendToAll(ctx, "Place Alert", body, fcmTokens)
}

func (n *Notifier) sendToAll(ctx context.Context, title, body string, tokens []string) {
	for _, token := range tokens {
		msg := Message{Token: token, Title: title, Body: body}
		if err := n.sender.Send(ctx, msg); err != nil {
			log.Printf("fcm send to %s: %v", token[:8], err)
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd server && go test ./internal/notify/ -v -count=1
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: FCM notification package with sender interface"
```

---

## Task 15: FCM Sender Implementation

**Files:**
- Create: `server/internal/notify/fcm.go`

- [ ] **Step 1: Implement real FCM sender**

Create `server/internal/notify/fcm.go`:

```go
package notify

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FCMSender struct {
	client *messaging.Client
}

func NewFCMSender(ctx context.Context, credentialsFile string) (*FCMSender, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("init firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("init messaging client: %w", err)
	}

	return &FCMSender{client: client}, nil
}

func (f *FCMSender) Send(ctx context.Context, msg Message) error {
	_, err := f.client.Send(ctx, &messaging.Message{
		Token: msg.Token,
		Notification: &messaging.Notification{
			Title: msg.Title,
			Body:  msg.Body,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ChannelID: "place_alerts",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("send fcm: %w", err)
	}
	return nil
}

// NoopSender is used when FCM credentials are not configured.
type NoopSender struct{}

func (n NoopSender) Send(ctx context.Context, msg Message) error {
	fmt.Printf("[noop-fcm] would send to %s: %s - %s\n", msg.Token, msg.Title, msg.Body)
	return nil
}
```

- [ ] **Step 2: Install Firebase dependency**

```bash
cd server && go get firebase.google.com/go/v4
cd server && go get google.golang.org/api
```

- [ ] **Step 3: Verify it compiles**

```bash
cd server && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: FCM sender implementation with noop fallback"
```

---

## Task 16: Wire Everything in main.go

**Files:**
- Modify: `server/cmd/tracker/main.go`
- Modify: `server/internal/api/server.go` (add WebSocket route, accept Hub)

- [ ] **Step 1: Update server.go to accept all dependencies and wire WS route**

Update `server/internal/api/server.go` — final version of `Server` struct and `NewServer`:

```go
type Server struct {
	router    chi.Router
	auth      *auth.Auth
	store     AuthStore
	locations LocationStore
	circles   CircleStore
	geofences GeofenceStore
	hub       *ws.Hub
}

func NewServer(
	a *auth.Auth,
	store AuthStore,
	locations LocationStore,
	circles CircleStore,
	geofences GeofenceStore,
	hub *ws.Hub,
) *Server {
	s := &Server{
		router:    chi.NewRouter(),
		auth:      a,
		store:     store,
		locations: locations,
		circles:   circles,
		geofences: geofences,
		hub:       hub,
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

	return s
}
```

Add the WebSocket handler to `server.go`:

```go
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	circleIDStr := r.URL.Query().Get("circle_id")
	if circleIDStr == "" {
		http.Error(w, "circle_id query param required", http.StatusBadRequest)
		return
	}

	circleID, err := uuid.Parse(circleIDStr)
	if err != nil {
		http.Error(w, "invalid circle_id", http.StatusBadRequest)
		return
	}

	s.hub.HandleConnect(w, r, userID, circleID)
}
```

Import `"github.com/nschatz/tracker/server/internal/ws"`.

- [ ] **Step 2: Write final main.go**

Update `server/cmd/tracker/main.go`:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nschatz/tracker/server/internal/api"
	"github.com/nschatz/tracker/server/internal/auth"
	"github.com/nschatz/tracker/server/internal/geo"
	"github.com/nschatz/tracker/server/internal/notify"
	"github.com/nschatz/tracker/server/internal/store"
	"github.com/nschatz/tracker/server/internal/ws"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	port := envOrDefault("PORT", "8080")
	dbURL := requireEnv("DATABASE_URL")
	jwtSecret := requireEnv("JWT_SECRET")
	fcmCreds := os.Getenv("FCM_CREDENTIALS_FILE")
	retentionDays := envIntOrDefault("LOCATION_RETENTION_DAYS", 30)

	// Database
	db, err := store.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	// Auth
	a := auth.New(jwtSecret)

	// WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// FCM notifications
	var sender notify.Sender
	if fcmCreds != "" {
		s, err := notify.NewFCMSender(ctx, fcmCreds)
		if err != nil {
			log.Fatalf("fcm: %v", err)
		}
		sender = s
	} else {
		log.Println("WARNING: FCM_CREDENTIALS_FILE not set, using noop sender")
		sender = notify.NoopSender{}
	}
	_ = notify.NewNotifier(sender)

	// Geofence tracker
	_ = geo.NewTracker()

	// API server
	srv := api.NewServer(a, db, db, db, db, hub)

	// Data retention
	go runRetention(ctx, db, retentionDays)

	// Start HTTP server
	httpSrv := &http.Server{
		Addr:    ":" + port,
		Handler: srv,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpSrv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on :%s", port)
	if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("http: %v", err)
	}
}

func runRetention(ctx context.Context, db interface{ DeleteLocationsOlderThan(context.Context, int) (int64, error) }, days int) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := db.DeleteLocationsOlderThan(ctx, days)
			if err != nil {
				log.Printf("retention: %v", err)
			} else if count > 0 {
				log.Printf("retention: deleted %d old location rows", count)
			}
		}
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func envIntOrDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("env var %s must be an integer: %v", key, err)
	}
	return n
}
```

- [ ] **Step 3: Verify everything compiles**

```bash
cd server && go build ./...
```

- [ ] **Step 4: Smoke test with Docker Compose**

```bash
docker compose up --build -d
sleep 3
curl http://localhost:8080/health
# Expected: "ok"
docker compose logs tracker-server | head -20
# Expected: migration applied, listening on :8080
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: wire all components in main.go with graceful shutdown and retention"
```

---

## Task 17: Integration — Geofence Evaluation on Location Ingestion

**Files:**
- Modify: `server/internal/api/location_handlers.go` (trigger geofence evaluation after insert)
- Modify: `server/internal/api/server.go` (add GeoEvaluator interface)

This wires the geo tracker and notifier into the location ingestion flow so that posting a location triggers geofence enter/leave detection and FCM notifications.

- [ ] **Step 1: Add evaluation interface to server.go**

Add to `server/internal/api/server.go`:

```go
type GeoEvaluator interface {
	FindContainingGeofences(ctx context.Context, circleID uuid.UUID, lat, lng float64) ([]uuid.UUID, error)
	GetGeofences(ctx context.Context, circleID uuid.UUID) ([]model.Geofence, error)
}
```

Add `geoTracker *geo.Tracker`, `notifier *notify.Notifier`, and `geoEval GeoEvaluator` to the `Server` struct. Update `NewServer` to accept them. Import `geo` and `notify` packages.

- [ ] **Step 2: Update handlePostLocations to trigger evaluation**

Update `server/internal/api/location_handlers.go` — after inserting locations, add:

```go
func (s *Server) handlePostLocations(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req postLocationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Locations) == 0 {
		http.Error(w, "at least one location required", http.StatusBadRequest)
		return
	}

	if err := s.locations.InsertLocations(r.Context(), userID, req.Locations); err != nil {
		http.Error(w, "failed to store locations", http.StatusInternalServerError)
		return
	}

	// Use the latest point for geofence evaluation and broadcast
	latest := req.Locations[len(req.Locations)-1]

	// Broadcast to WebSocket clients
	loc := model.Location{
		UserID: userID, Lat: latest.Lat, Lng: latest.Lng,
		Speed: latest.Speed, BatteryLevel: latest.BatteryLevel,
		Accuracy: latest.Accuracy, RecordedAt: latest.RecordedAt,
	}

	// Evaluate geofences for each circle the user is in
	if s.geoTracker != nil && s.geoEval != nil {
		go s.evaluateGeofences(r.Context(), userID, latest.Lat, latest.Lng, loc)
	} else {
		// Still broadcast even without geofence evaluation
		circles, _ := s.circles.GetUserCircles(r.Context(), userID)
		for _, c := range circles {
			s.hub.BroadcastLocation(c.ID, loc)
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) evaluateGeofences(ctx context.Context, userID uuid.UUID, lat, lng float64, loc model.Location) {
	circles, err := s.circles.GetUserCircles(ctx, userID)
	if err != nil {
		log.Printf("get user circles: %v", err)
		return
	}

	for _, circle := range circles {
		s.hub.BroadcastLocation(circle.ID, loc)

		containing, err := s.geoEval.FindContainingGeofences(ctx, circle.ID, lat, lng)
		if err != nil {
			log.Printf("find containing geofences: %v", err)
			continue
		}

		entered, left := s.geoTracker.Update(userID, containing)

		if len(entered) == 0 && len(left) == 0 {
			continue
		}

		// Get user display name
		user, err := s.store.GetUserByID(ctx, userID)
		if err != nil {
			log.Printf("get user: %v", err)
			continue
		}

		// Get geofence names for notifications
		geofences, err := s.geoEval.GetGeofences(ctx, circle.ID)
		if err != nil {
			log.Printf("get geofences: %v", err)
			continue
		}
		gfNames := make(map[uuid.UUID]string)
		for _, gf := range geofences {
			gfNames[gf.ID] = gf.Name
		}

		// Get FCM tokens for other circle members
		// (FCM token storage is not yet implemented — this is a placeholder
		// that will be completed in a follow-up task when the Android app
		// registers its FCM token with the server)
		_ = user
		_ = gfNames
		_ = entered
		_ = left
	}
}
```

Add `"log"` to imports. Add `GetUserByID` to the `AuthStore` interface:

```go
type AuthStore interface {
	CreateUser(ctx context.Context, email, displayName, passwordHash string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetCircleByInviteCode(ctx context.Context, code string) (*model.Circle, error)
	AddMember(ctx context.Context, circleID, userID uuid.UUID, role string) error
}
```

- [ ] **Step 3: Update main.go to pass geo tracker and notifier to server**

Update the `api.NewServer` call in `main.go` to pass the tracker and notifier:

```go
geoTracker := geo.NewTracker()
notifier := notify.NewNotifier(sender)

srv := api.NewServer(a, db, db, db, db, hub, geoTracker, notifier, db)
```

Update `NewServer` signature accordingly.

- [ ] **Step 4: Verify everything compiles and tests pass**

```bash
cd server && go build ./...
cd server && go test ./... -count=1
# Expected: all tests PASS (update mock stores in test files for new NewServer signature)
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: wire geofence evaluation and WS broadcast into location ingestion"
```

---

## Task 18: FCM Token Registration Endpoint

**Files:**
- Create: `server/internal/store/fcm_tokens.go`
- Modify: `server/internal/store/migrations/001_initial.sql` (add fcm_tokens table — or create `002_fcm_tokens.sql`)
- Create: `server/internal/api/fcm_handlers.go`

- [ ] **Step 1: Create migration for FCM tokens table**

Create `server/internal/store/migrations/002_fcm_tokens.sql`:

```sql
CREATE TABLE fcm_tokens (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id)
);
```

- [ ] **Step 2: Implement FCM token store operations**

Create `server/internal/store/fcm_tokens.go`:

```go
package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) UpsertFCMToken(ctx context.Context, userID uuid.UUID, token string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO fcm_tokens (user_id, token)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET token = $2, updated_at = now()
	`, userID, token)
	if err != nil {
		return fmt.Errorf("upsert fcm token: %w", err)
	}
	return nil
}

func (s *Store) GetFCMTokensForCircle(ctx context.Context, circleID uuid.UUID, excludeUserID uuid.UUID) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ft.token
		FROM fcm_tokens ft
		JOIN circle_members cm ON cm.user_id = ft.user_id
		WHERE cm.circle_id = $1 AND ft.user_id != $2
	`, circleID, excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("query fcm tokens: %w", err)
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, fmt.Errorf("scan token: %w", err)
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}
```

- [ ] **Step 3: Add FCM token endpoint**

Create `server/internal/api/fcm_handlers.go`:

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/nschatz/tracker/server/internal/auth"
)

type registerTokenRequest struct {
	Token string `json:"token"`
}

func (s *Server) handleRegisterFCMToken(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req registerTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}

	if err := s.fcmTokens.UpsertFCMToken(r.Context(), userID, req.Token); err != nil {
		http.Error(w, "failed to register token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
```

Add `FCMTokenStore` interface to `server.go`:

```go
type FCMTokenStore interface {
	UpsertFCMToken(ctx context.Context, userID uuid.UUID, token string) error
	GetFCMTokensForCircle(ctx context.Context, circleID uuid.UUID, excludeUserID uuid.UUID) ([]string, error)
}
```

Add `fcmTokens FCMTokenStore` to `Server` struct. Add route: `r.Post("/fcm-token", s.handleRegisterFCMToken)`. Update `NewServer` signature.

- [ ] **Step 4: Wire FCM tokens into geofence evaluation**

Update `evaluateGeofences` in `location_handlers.go` to replace the placeholder with actual notifications:

```go
		// Get FCM tokens for other circle members
		tokens, err := s.fcmTokens.GetFCMTokensForCircle(ctx, circle.ID, userID)
		if err != nil {
			log.Printf("get fcm tokens: %v", err)
			continue
		}

		for _, gfID := range entered {
			if name, ok := gfNames[gfID]; ok {
				s.notifier.GeofenceEnter(ctx, user.DisplayName, name, tokens)
			}
		}
		for _, gfID := range left {
			if name, ok := gfNames[gfID]; ok {
				s.notifier.GeofenceLeave(ctx, user.DisplayName, name, tokens)
			}
		}
```

- [ ] **Step 5: Verify everything compiles**

```bash
cd server && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: FCM token registration and geofence notification delivery"
```

---

## Task 19: End-to-End Smoke Test

**Files:**
- No new files — this is a manual verification task

- [ ] **Step 1: Start fresh**

```bash
docker compose down -v
docker compose up --build -d
sleep 5
```

- [ ] **Step 2: Create a user and circle via curl**

Since registration requires an invite code, first create a user + circle directly in the DB, then test the API flow:

```bash
# First user registers without invite (we need to create the first circle)
# Add a bootstrap endpoint or seed the DB. For now, use psql:
docker compose exec postgres psql -U tracker -c "
INSERT INTO users (id, email, display_name, password_hash)
VALUES ('00000000-0000-0000-0000-000000000001', 'admin@test.com', 'Admin',
        '\$2a\$10\$dummy');
INSERT INTO circles (id, name, invite_code, created_by)
VALUES ('00000000-0000-0000-0000-000000000010', 'Family', 'INVITE1',
        '00000000-0000-0000-0000-000000000001');
INSERT INTO circle_members (circle_id, user_id, role)
VALUES ('00000000-0000-0000-0000-000000000010',
        '00000000-0000-0000-0000-000000000001', 'admin');
"

# Register second user with invite code
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@test.com","display_name":"User","password":"test123","invite_code":"INVITE1"}'
# Expected: 201 with token
```

- [ ] **Step 3: Post a location and query it**

```bash
# Use the token from registration
TOKEN="<token from step 2>"

curl -s -X POST http://localhost:8080/locations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"locations":[{"lat":40.7128,"lng":-74.0060,"recorded_at":"2026-04-01T12:00:00Z"}]}'
# Expected: 202

curl -s "http://localhost:8080/locations/latest?circle_id=00000000-0000-0000-0000-000000000010" \
  -H "Authorization: Bearer $TOKEN"
# Expected: JSON array with the location
```

- [ ] **Step 4: Verify Docker Compose logs show no errors**

```bash
docker compose logs tracker-server --tail 50
# Expected: migrations applied, no panics or errors
```

- [ ] **Step 5: Commit (no code changes — just verify)**

No commit needed if no fixes were required. If fixes were made, commit them:

```bash
git add -A
git commit -m "fix: address issues found during smoke test"
```

---

## Summary

| Task | Component | What it builds |
|------|-----------|---------------|
| 1 | Scaffolding | Go module, Docker Compose, Dockerfile, health endpoint |
| 2 | Database | PostGIS schema, migration runner |
| 3 | Model | Shared types (User, Circle, Location, Geofence) |
| 4 | Store | User + circle DB operations |
| 5 | Auth | JWT, bcrypt, HTTP middleware |
| 6 | API | Router + auth endpoints (register/login) |
| 7 | Store | Location insert, latest, history, retention |
| 8 | API | Location endpoints |
| 9 | WebSocket | Hub with broadcast + connection management |
| 10 | API | Circle CRUD endpoints |
| 11 | Store | Geofence CRUD + PostGIS spatial queries |
| 12 | API | Geofence CRUD endpoints |
| 13 | Geo | In-memory geofence state tracker |
| 14 | Notify | Notification package with sender interface |
| 15 | Notify | FCM sender implementation |
| 16 | Main | Wire all components, graceful shutdown, retention |
| 17 | Integration | Geofence eval + WS broadcast on location ingestion |
| 18 | FCM Tokens | Token registration + notification delivery |
| 19 | Smoke Test | End-to-end verification |
