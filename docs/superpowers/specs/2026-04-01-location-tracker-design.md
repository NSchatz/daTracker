# Location Tracker — Design Spec

A self-hosted Life360 replacement for real-time location sharing, geofence alerts, and location history within a small group (2-10 users). Android-only.

## System Architecture

Three components deployed via Docker Compose:

1. **Go API server** — auth, location ingestion, geofence evaluation, WebSocket broadcast, FCM push
2. **PostgreSQL + PostGIS** — persistent storage with geospatial indexing
3. **Kotlin Android app** — adaptive background location tracking, OSM map display, real-time updates

### Data Flow

1. Phone detects movement and sends location update(s) via HTTPS POST to server
2. Server persists to PostGIS, evaluates geofence triggers, broadcasts to connected WebSocket clients
3. If a geofence boundary is crossed and the target user is not connected via WebSocket, send FCM push notification
4. App displays live map with all circle members' positions via WebSocket stream
5. Location history queries hit PostGIS with spatial + temporal indexes

## Data Model (PostgreSQL + PostGIS)

### users
| Column | Type | Notes |
|--------|------|-------|
| `id` | `UUID` | PK |
| `email` | `TEXT UNIQUE` | Login identifier |
| `display_name` | `TEXT` | Shown on map |
| `password_hash` | `TEXT` | bcrypt |
| `created_at` | `TIMESTAMPTZ` | |

### circles
| Column | Type | Notes |
|--------|------|-------|
| `id` | `UUID` | PK |
| `name` | `TEXT` | Circle display name |
| `invite_code` | `TEXT UNIQUE` | For joining |
| `created_by` | `UUID` | FK -> users |
| `created_at` | `TIMESTAMPTZ` | |

### circle_members
| Column | Type | Notes |
|--------|------|-------|
| `circle_id` | `UUID` | FK -> circles |
| `user_id` | `UUID` | FK -> users |
| `role` | `TEXT` | 'admin' or 'member' |
| `joined_at` | `TIMESTAMPTZ` | |

PK: (`circle_id`, `user_id`)

### locations
| Column | Type | Notes |
|--------|------|-------|
| `id` | `BIGSERIAL` | PK |
| `user_id` | `UUID` | FK -> users |
| `point` | `GEOGRAPHY(Point, 4326)` | PostGIS point |
| `speed` | `REAL` | m/s |
| `battery_level` | `SMALLINT` | 0-100 |
| `accuracy` | `REAL` | meters |
| `recorded_at` | `TIMESTAMPTZ` | When the phone recorded the location |

Append-only table. Indexes: GIST on `point`, BRIN on `recorded_at`, B-tree on `user_id + recorded_at`.

### geofences
| Column | Type | Notes |
|--------|------|-------|
| `id` | `UUID` | PK |
| `circle_id` | `UUID` | FK -> circles |
| `name` | `TEXT` | e.g. "Home", "Work" |
| `center` | `GEOGRAPHY(Point, 4326)` | PostGIS point |
| `radius_meters` | `REAL` | Geofence radius |
| `created_by` | `UUID` | FK -> users |
| `created_at` | `TIMESTAMPTZ` | |

Geofences are circles (center + radius). Evaluated with `ST_DWithin()`.

### Data Retention

A background goroutine runs daily and deletes location rows older than `LOCATION_RETENTION_DAYS` (default: 30). Users, circles, and geofences are not subject to retention.

## Go API Server

### Package Structure

| Package | Responsibility |
|---------|---------------|
| `api` | HTTP handlers + routing (`net/http` + `chi` router) |
| `ws` | WebSocket hub — manages connections, broadcasts location updates to circle members |
| `geo` | Geofence evaluation — checks new locations against active geofences via PostGIS |
| `notify` | FCM push notifications for geofence enter/leave alerts |
| `auth` | JWT issuance/validation, invite code generation, password hashing (bcrypt) |
| `store` | Database access layer (PostGIS queries, location inserts, user/circle CRUD) |

### API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/auth/register` | Create account (requires invite code) |
| `POST` | `/auth/login` | Returns JWT token |
| `POST` | `/locations` | Submit location update (supports batches) |
| `GET` | `/locations/history` | Query location history for a single user (user_id, time range) |
| `GET` | `/locations/latest` | Latest position for all circle members |
| `GET` | `/ws` | WebSocket upgrade — real-time location stream |
| `POST` | `/circles` | Create circle |
| `POST` | `/circles/:id/join` | Join circle with invite code |
| `GET` | `/circles/:id/members` | List circle members |
| `POST` | `/geofences` | Create geofence |
| `GET` | `/geofences` | List geofences for circle |
| `PUT` | `/geofences/:id` | Update geofence |
| `DELETE` | `/geofences/:id` | Delete geofence |

### Location Ingestion Flow

1. Phone POSTs batch of location points
2. Server bulk-inserts into `locations` table
3. Server runs `ST_DWithin()` against all circle geofences for the newest point
4. Compare against in-memory geofence state (`user_id -> set of geofence_ids currently inside`) to detect enter/leave transitions
5. On transition: send FCM notification to relevant circle members
6. Broadcast latest position to WebSocket hub for all connected circle members

### Geofence State Tracking

The server maintains an in-memory map: `user_id -> set of geofence_ids they're currently inside`. On each location update, it computes the new set and compares against the old to detect enter/leave transitions. This map is rebuilt from the latest positions on server startup.

## Android App (Kotlin)

### Location Tracking (Foreground Service)

- **Fused Location Provider** with adaptive intervals based on Activity Recognition API:
  - Stationary: every 5 minutes
  - Walking: every 30 seconds
  - Driving: every 10 seconds
- Runs as a **foreground service** with persistent notification (required by Android for reliable background location)
- Batches location points and sends to server on each update, or every 30 seconds, whichever comes first
- If network is unavailable, queues updates in a local **Room database** and flushes when connectivity returns

### Map UI

- **osmdroid** for OpenStreetMap tile rendering
- Circle members shown as markers with name, last update time, and battery level
- Markers update in real-time via WebSocket
- Tap a member to see their location history as a path on the map
- Geofences rendered as translucent circles, tap to edit/delete

### Screens

| Screen | Purpose |
|--------|---------|
| Map (main) | Live map with all members' positions |
| History | Select member + date range, view path on map |
| Places | List/create/edit geofences with map picker |
| Circle | Member list, invite link sharing, settings |
| Profile | Display name, notification preferences |

### Notifications

- **FCM** for geofence enter/leave alerts when app is backgrounded
- **WebSocket** for real-time updates when app is foregrounded
- Notification channels: "Place Alerts" (geofence triggers), "App Status" (foreground service)

### Offline Behavior

- Map tiles cached locally by osmdroid
- Last known positions cached in Room DB — app shows stale data with timestamps rather than empty map
- Location tracking continues regardless of server connectivity

## Deployment

### Docker Compose Services

| Service | Image | Notes |
|---------|-------|-------|
| `tracker-server` | Custom Go build (~15MB) | Single binary |
| `postgres` | `postgis/postgis:16-3.4` | Persistent volume |

### Configuration (Environment Variables)

| Variable | Purpose | Default |
|----------|---------|---------|
| `DATABASE_URL` | PostGIS connection string | required |
| `JWT_SECRET` | Token signing key | required |
| `FCM_CREDENTIALS_FILE` | Path to Firebase service account JSON | required |
| `LOCATION_RETENTION_DAYS` | How long to keep history | `30` |
| `WS_PING_INTERVAL` | WebSocket keepalive interval | `30s` |
| `PORT` | HTTP listen port | `8080` |

### Prerequisites

- **Firebase project** (free tier) for FCM credentials. No other Firebase services used.
- **Reverse proxy** (Caddy, nginx, or Traefik) for HTTPS termination. Not included in Compose file. Required for both REST API and WebSocket connections.

### Backup

PostgreSQL volume backed up with `pg_dump` or volume snapshots. Location data is the only table that grows significantly.
