# FCM to ntfy Migration — Design Spec

Replace Firebase Cloud Messaging with self-hosted ntfy for push notifications, eliminating the Google/Firebase dependency.

## Overview

ntfy is a simple HTTP-based pub/sub notification service. Each user gets a unique topic (`tracker-{userId}`). The Go server POSTs notifications to ntfy, and the Android app subscribes to its topic via SSE (Server-Sent Events) from the existing foreground service.

## Backend Changes

### Docker Compose

Add ntfy service:
```yaml
ntfy:
  image: binwiederhier/ntfy
  command: serve
  ports:
    - "8090:80"
  volumes:
    - ntfy-cache:/var/cache/ntfy
```

### notify/ntfy.go (replaces fcm.go)

`NtfySender` implements the existing `Sender` interface:
- Constructor takes ntfy server URL (e.g., `http://ntfy:80`)
- `Send()` makes HTTP POST to `{ntfyURL}/tracker-{token}` where token is the user ID
- Request body: JSON with `title` and `message` fields, `priority` set to `high`

`NoopSender` stays as-is for when ntfy is not configured.

### Remove Firebase

- Delete `notify/fcm.go`
- Remove `firebase.google.com/go/v4` and related GCP dependencies from `go.mod`
- Env var: `FCM_CREDENTIALS_FILE` replaced by `NTFY_URL` (default: `http://ntfy:80`)

### Remove token management

Since ntfy uses topics derived from user IDs, device token registration is unnecessary:
- Delete `store/fcm_tokens.go`
- Delete `api/fcm_handlers.go`
- Remove `/fcm-token` route from `server.go`
- Remove `FCMTokenStore` interface from `server.go`
- Create migration `003_drop_fcm_tokens.sql`: `DROP TABLE IF EXISTS fcm_tokens`

### Update notification delivery

In `location_handlers.go`, the geofence notification flow changes:
- Old: get FCM tokens for circle members, send to each token
- New: get user IDs for circle members, send to each user's topic (`tracker-{userId}`)

The `Notifier.GeofenceEnter/Leave` methods change signature: accept `[]string` of user IDs instead of FCM tokens. The `Sender.Send` `Message.Token` field becomes the user ID (used as the ntfy topic suffix).

## Android Changes

### Remove Firebase

- Remove `com.google.gms.google-services` plugin from `app/build.gradle.kts`
- Remove `com.google.gms:google-services` from project `build.gradle.kts`
- Remove `firebase-bom` and `firebase-messaging` dependencies
- Delete `fcm/TrackerFirebaseService.kt`
- Remove Firebase service from `AndroidManifest.xml`
- Remove `FcmApi.kt` and `FcmTokenRequest` from `ApiModels.kt`
- Remove `fcm` property from `ApiClient.kt`
- Remove FCM token registration from `MainActivity.kt`

### Add ntfy SSE listener to LocationService

The existing `LocationService` foreground service adds an SSE subscription:
- On start: open persistent HTTP connection to `{ntfyUrl}/tracker-{userId}/sse` (ntfy URL stored in SessionManager, defaults to server URL on port 8090)
- Parse incoming JSON messages (each line is a JSON object with `title`, `message`, `event`)
- On message received: display local notification on `CHANNEL_PLACE_ALERTS`
- Auto-reconnect on connection failure (5s delay)
- Disconnect on service stop

The SSE connection uses OkHttp (already a dependency) with a streaming response body.

### SessionManager

Add a helper: `ntfyTopic` computed property returning `"tracker-${userId}"`.

## Configuration

| Variable | Purpose | Default |
|----------|---------|---------|
| `NTFY_URL` | ntfy server URL | `http://ntfy:80` |

`FCM_CREDENTIALS_FILE` is removed entirely.

## What stays the same

- `Sender` interface in `notify/notify.go`
- `Notifier` struct and its `GeofenceEnter`/`GeofenceLeave` methods (signature changes slightly)
- `CHANNEL_PLACE_ALERTS` notification channel on Android
- Notification display logic (title, body, tap to open MainActivity)
