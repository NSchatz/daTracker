# Web App — Design Spec

A browser-based frontend for the location tracker with full feature parity to the Android app. Vanilla HTML/JS with Leaflet.js for maps, embedded in the Go binary as static files.

## Architecture

Static files in `server/web/` embedded via Go's `embed.FS`. The Go server serves them as a catch-all for non-API routes. Hash-based SPA routing (`#map`, `#history`, etc.). No build step, no Node.js.

### File Structure

```
server/web/
  index.html              - Single page shell with nav bar
  css/
    style.css             - All styles
  js/
    app.js                - Router, auth state, API client, nav management
    auth.js               - Login + register forms
    map.js                - Leaflet map, member markers, geofence circles, WebSocket
    history.js            - Path display with member/date selector
    places.js             - Geofence list, create/edit with map picker
    circle.js             - Member list, invite code display
    profile.js            - Settings, logout
  lib/
    leaflet.js            - Vendored Leaflet JS
    leaflet.css           - Vendored Leaflet CSS
    images/               - Leaflet marker images
```

### Go Server Changes

- New file `server/internal/api/static.go` — embeds `web/` directory, serves static files
- Modify `server/internal/api/server.go` — add catch-all file server handler after API routes. Any path that doesn't match an API route serves from embedded files. Unknown paths return `index.html` for SPA routing.
- No new API endpoints. The web app uses the same REST API and WebSocket as the Android app.

## External Dependencies

- **Leaflet.js 1.9.4** — vendored in `web/lib/` (no CDN dependency for self-hosting)

No other frontend dependencies.

## Screens

### Login (`#login`)

- Email + password fields
- Server URL field (pre-filled from localStorage, defaults to current origin)
- Login button → `POST /auth/login` → stores JWT + user info in localStorage
- Link to register screen
- Error display on failure

### Register (`#register`)

- Display name, email, password, invite code fields
- Register button → `POST /auth/register` → stores JWT, navigates to `#map`
- Error display on failure

### Map (`#map`) — default screen

- Full-screen Leaflet map with OSM tiles
- Member markers showing name, last update time, battery level in popup
- Geofences rendered as translucent blue circles with name labels
- WebSocket connection to `/ws?circle_id={id}` for real-time marker updates
- Auto-centers on first load to fit all members
- WebSocket connects on enter, disconnects on leave

### History (`#history`)

- Member dropdown (populated from `GET /circles/{id}/members`)
- Date picker (HTML native `<input type="date">`, defaults to today)
- Leaflet map showing polyline path for selected member + date
- Path drawn in blue, map auto-fits to path bounds
- Clears previous path when selection changes

### Places (`#places`)

- List of geofences from `GET /geofences?circle_id={id}`
- Each item shows name + radius, with delete button
- "Add Place" button opens a modal/inline form:
  - Name field
  - Radius field (default 100m)
  - Leaflet map for click-to-place center point
  - Marker follows clicks
  - Save → `POST /geofences`, refresh list

### Circle (`#circle`)

- Circle name display
- Invite code with copy-to-clipboard button
- Member list from `GET /circles/{id}/members` showing display name + role

### Profile (`#profile`)

- Display name + email (read-only, from localStorage)
- Server URL field (editable, saves to localStorage)
- Logout button → clears localStorage, navigates to `#login`

## Navigation

Top nav bar (desktop-optimized). Tabs: Map, History, Places, Circle, Profile. Highlights active tab. Hidden when not authenticated (login/register screens are full-page).

## Auth Flow

1. On page load, check localStorage for `token`
2. If missing, navigate to `#login`
3. All API calls include `Authorization: Bearer {token}` header
4. On 401 response from any API call, clear token and redirect to `#login`
5. JWT stored in localStorage (persists across tabs/sessions)
6. `activeCircleId` stored in localStorage — fetched from `GET /circles` on first login if not set

## WebSocket

- Connects to `ws://{host}/ws?circle_id={id}` (or `wss://` for HTTPS)
- Auth via query param or protocol header (WebSocket doesn't support custom headers in browsers — pass token as query param: `/ws?circle_id={id}&token={jwt}`)
- Receives JSON `MemberLocation` objects, updates corresponding marker on map
- Auto-reconnect on disconnect (5s delay)
- Only active on map screen

**Backend change needed:** The current WebSocket handler reads JWT from the `Authorization` header. Browsers can't set custom headers on WebSocket connections. The handler needs to also accept `?token=` query parameter as a fallback. This is a one-line change in `handleWebSocket`.

## Styling

- Clean, minimal dark nav bar with light content area
- Blue primary color (#1B73E8) matching Android app
- Desktop-optimized — no mobile breakpoints needed
- Map takes full remaining viewport height below nav bar
