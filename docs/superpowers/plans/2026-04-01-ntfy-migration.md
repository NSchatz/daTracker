# FCM to ntfy Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Firebase Cloud Messaging with self-hosted ntfy for push notifications, removing all Google/Firebase dependencies.

**Architecture:** ntfy is added as a Docker Compose service. The Go server POSTs to ntfy topics (one per user: `tracker-{userId}`). The Android app subscribes to its topic via SSE from the existing foreground service. No device token exchange needed — user ID is the addressing mechanism.

**Tech Stack:** ntfy (binwiederhier/ntfy), Go stdlib net/http for ntfy POST, OkHttp SSE for Android subscription

---

## File Changes Overview

### Go Backend — Modify
- `server/internal/notify/fcm.go` → **Delete** (replaced by ntfy.go)
- `server/internal/notify/ntfy.go` → **Create** (NtfySender implementing Sender interface)
- `server/internal/notify/notify.go` → **Modify** (rename params from fcmTokens to userIDs)
- `server/internal/notify/notify_test.go` → **Modify** (update param names)
- `server/internal/store/fcm_tokens.go` → **Delete**
- `server/internal/store/migrations/002_fcm_tokens.sql` → **Keep** (already applied)
- `server/internal/store/migrations/003_drop_fcm_tokens.sql` → **Create**
- `server/internal/api/fcm_handlers.go` → **Delete**
- `server/internal/api/server.go` → **Modify** (remove FCMTokenStore, remove /fcm-token route, update NewServer)
- `server/internal/api/location_handlers.go` → **Modify** (get member user IDs instead of FCM tokens)
- `server/internal/api/auth_handlers_test.go` → **Modify** (update NewServer call)
- `server/internal/api/circle_handlers_test.go` → **Modify** (update NewServer call)
- `server/internal/api/location_handlers_test.go` → **Modify** (update NewServer call)
- `server/cmd/tracker/main.go` → **Modify** (NTFY_URL instead of FCM_CREDENTIALS_FILE)
- `server/go.mod` → **Modify** (remove firebase dependencies)
- `docker-compose.yml` → **Modify** (add ntfy service, remove FCM env var)
- `.env.example` → **Modify** (NTFY_URL instead of FCM_CREDENTIALS_FILE)

### Android — Modify
- `android/app/build.gradle.kts` → **Modify** (remove Firebase deps)
- `android/app/src/main/AndroidManifest.xml` → **Modify** (remove Firebase service)
- `android/app/src/main/java/com/nschatz/tracker/fcm/TrackerFirebaseService.kt` → **Delete**
- `android/app/src/main/java/com/nschatz/tracker/data/api/FcmApi.kt` → **Delete**
- `android/app/src/main/java/com/nschatz/tracker/data/api/ApiClient.kt` → **Modify** (remove fcm property)
- `android/app/src/main/java/com/nschatz/tracker/data/model/ApiModels.kt` → **Modify** (remove FcmTokenRequest)
- `android/app/src/main/java/com/nschatz/tracker/data/prefs/SessionManager.kt` → **Modify** (add ntfyUrl)
- `android/app/src/main/java/com/nschatz/tracker/service/LocationService.kt` → **Modify** (add SSE listener)
- `android/app/src/main/java/com/nschatz/tracker/ui/main/MainActivity.kt` → **Modify** (remove FCM token registration)

---

## Task 1: Go Backend — Replace FCM Sender with ntfy Sender

**Files:**
- Delete: `server/internal/notify/fcm.go`
- Create: `server/internal/notify/ntfy.go`
- Modify: `server/internal/notify/notify.go`
- Modify: `server/internal/notify/notify_test.go`

- [ ] **Step 1: Write ntfy sender test**

Add to `server/internal/notify/notify_test.go`:

```go
func TestNtfySender(t *testing.T) {
	// Start a test HTTP server that records requests
	var received []struct {
		topic string
		title string
		body  string
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = append(received, struct {
			topic string
			title string
			body  string
		}{
			topic: r.URL.Path,
			title: r.Header.Get("Title"),
			body:  string(body),
		})
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sender, err := NewNtfySender(ts.URL)
	if err != nil {
		t.Fatalf("NewNtfySender: %v", err)
	}

	ctx := context.Background()
	err = sender.Send(ctx, Message{Token: "user-123", Title: "Place Alert", Body: "Alice arrived"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if len(received) != 1 {
		t.Fatalf("expected 1 request, got %d", len(received))
	}
	if received[0].topic != "/tracker-user-123" {
		t.Errorf("topic = %q, want /tracker-user-123", received[0].topic)
	}
	if received[0].title != "Place Alert" {
		t.Errorf("title = %q, want Place Alert", received[0].title)
	}
	if received[0].body != "Alice arrived" {
		t.Errorf("body = %q, want Alice arrived", received[0].body)
	}
}
```

Add imports: `"io"`, `"net/http"`, `"net/http/httptest"`.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd server && go test ./internal/notify/ -run TestNtfySender -v -count=1
# Expected: FAIL — NewNtfySender not defined
```

- [ ] **Step 3: Create ntfy.go**

Create `server/internal/notify/ntfy.go`:

```go
package notify

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// NtfySender sends notifications via ntfy HTTP API.
type NtfySender struct {
	baseURL string
	client  *http.Client
}

func NewNtfySender(baseURL string) (*NtfySender, error) {
	return &NtfySender{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}, nil
}

func (n *NtfySender) Send(ctx context.Context, msg Message) error {
	topic := fmt.Sprintf("tracker-%s", msg.Token)
	url := fmt.Sprintf("%s/%s", n.baseURL, topic)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(msg.Body))
	if err != nil {
		return fmt.Errorf("ntfy: create request: %w", err)
	}
	req.Header.Set("Title", msg.Title)
	req.Header.Set("Priority", "high")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// NoopSender is used when ntfy is not configured.
type NoopSender struct{}

func (n NoopSender) Send(ctx context.Context, msg Message) error {
	fmt.Printf("[noop-ntfy] topic=tracker-%s title=%q body=%q\n", msg.Token, msg.Title, msg.Body)
	return nil
}
```

- [ ] **Step 4: Delete fcm.go**

```bash
rm server/internal/notify/fcm.go
```

- [ ] **Step 5: Update notify.go — rename params from fcmTokens to userIDs**

In `server/internal/notify/notify.go`, change:

```go
func (n *Notifier) GeofenceEnter(ctx context.Context, userName, placeName string, userIDs []string) {
	body := fmt.Sprintf("%s arrived at %s", userName, placeName)
	for _, userID := range userIDs {
		msg := Message{
			Token: userID,
			Title: "Place Alert",
			Body:  body,
		}
		if err := n.sender.Send(ctx, msg); err != nil {
			log.Printf("notify: GeofenceEnter send error for user %s: %v", userID, err)
		}
	}
}

func (n *Notifier) GeofenceLeave(ctx context.Context, userName, placeName string, userIDs []string) {
	body := fmt.Sprintf("%s left %s", userName, placeName)
	for _, userID := range userIDs {
		msg := Message{
			Token: userID,
			Title: "Place Alert",
			Body:  body,
		}
		if err := n.sender.Send(ctx, msg); err != nil {
			log.Printf("notify: GeofenceLeave send error for user %s: %v", userID, err)
		}
	}
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd server && go test ./internal/notify/ -v -count=1
# Expected: PASS — TestGeofenceEnter, TestGeofenceLeave, TestNtfySender
```

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat: replace FCM sender with ntfy HTTP sender"
```

---

## Task 2: Go Backend — Remove FCM Token Store + Handler, Update Server Wiring

**Files:**
- Delete: `server/internal/store/fcm_tokens.go`
- Delete: `server/internal/api/fcm_handlers.go`
- Create: `server/internal/store/migrations/003_drop_fcm_tokens.sql`
- Modify: `server/internal/api/server.go`
- Modify: `server/internal/api/location_handlers.go`
- Modify: `server/internal/api/auth_handlers_test.go`
- Modify: `server/internal/api/circle_handlers_test.go`
- Modify: `server/internal/api/location_handlers_test.go`

- [ ] **Step 1: Create drop migration**

Create `server/internal/store/migrations/003_drop_fcm_tokens.sql`:

```sql
DROP TABLE IF EXISTS fcm_tokens;
```

- [ ] **Step 2: Delete FCM files**

```bash
rm server/internal/store/fcm_tokens.go
rm server/internal/api/fcm_handlers.go
```

- [ ] **Step 3: Update server.go — remove FCMTokenStore, simplify NewServer**

Remove the `FCMTokenStore` interface entirely. Remove `fcmTokens FCMTokenStore` from Server struct. Remove it from `NewServer` params. Remove the `/fcm-token` route. The new `NewServer` signature:

```go
func NewServer(a *auth.Auth, store AuthStore, circles CircleStore, locations LocationStore, geofences GeofenceStore, hub *ws.Hub, geoTracker *geo.Tracker, notifier *notify.Notifier, geoEval GeoEvaluator) *Server {
```

(9 params instead of 10 — `fcmTokens` removed)

Remove `fcmTokens` from the struct initialization inside NewServer.

- [ ] **Step 4: Update location_handlers.go — get member user IDs instead of FCM tokens**

In `processLocationUpdate`, replace the FCM token section (lines 116-136) with:

```go
		// Get member user IDs for notifications (excluding the current user)
		members, err := s.circles.GetMembers(ctx, circle.ID)
		if err != nil {
			log.Printf("processLocationUpdate: get members for circle %s: %v", circle.ID, err)
			continue
		}

		var memberIDs []string
		for _, m := range members {
			if m.UserID != userID {
				memberIDs = append(memberIDs, m.UserID.String())
			}
		}
		if len(memberIDs) == 0 {
			continue
		}

		for _, geoID := range entered {
			name := geoMap[geoID]
			s.notifier.GeofenceEnter(ctx, user.DisplayName, name, memberIDs)
		}
		for _, geoID := range left {
			name := geoMap[geoID]
			s.notifier.GeofenceLeave(ctx, user.DisplayName, name, memberIDs)
		}
```

Remove the `if s.fcmTokens == nil` check. The nil check for `s.notifier` earlier in the function already gates this.

- [ ] **Step 5: Update all test files — NewServer now takes 9 args instead of 10**

In `auth_handlers_test.go`, `circle_handlers_test.go`, and `location_handlers_test.go`, change all `NewServer(...)` calls to remove the last `nil` argument:

```go
// Old (10 args):
srv := NewServer(authSvc, authStore, circleStore, nil, nil, nil, nil, nil, nil, nil)
// New (9 args):
srv := NewServer(authSvc, authStore, circleStore, nil, nil, nil, nil, nil, nil)
```

- [ ] **Step 6: Verify build and tests pass**

```bash
cd server && go build ./...
cd server && go test ./... -count=1 -timeout 30s
# Expected: all tests PASS
```

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat: remove FCM token store and handler, use member IDs for ntfy topics"
```

---

## Task 3: Go Backend — Update main.go, Docker Compose, env config

**Files:**
- Modify: `server/cmd/tracker/main.go`
- Modify: `docker-compose.yml`
- Modify: `.env.example`

- [ ] **Step 1: Update main.go**

Replace the FCM sender initialization (lines 28-51) with:

```go
	ntfyURL := envOrDefault("NTFY_URL", "http://ntfy:80")

	// ...existing db, auth, hub setup...

	var sender notify.Sender
	if ntfyURL != "" {
		s, err := notify.NewNtfySender(ntfyURL)
		if err != nil {
			log.Fatalf("ntfy: %v", err)
		}
		sender = s
		log.Printf("ntfy sender configured: %s", ntfyURL)
	} else {
		log.Println("WARNING: NTFY_URL not set, using noop sender")
		sender = notify.NoopSender{}
	}
```

Remove the `fcmCreds` variable. Update the `NewServer` call to remove the last `db` argument (was FCMTokenStore):

```go
srv := api.NewServer(a, db, db, db, db, hub, geoTracker, notifier, db)
```

(9 args instead of 10)

- [ ] **Step 2: Update docker-compose.yml**

Add ntfy service and update tracker-server environment:

```yaml
services:
  postgres:
    # ...unchanged...

  ntfy:
    image: binwiederhier/ntfy
    command: serve
    ports:
      - "8090:80"
    volumes:
      - ntfy-cache:/var/cache/ntfy

  tracker-server:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://tracker:tracker@postgres:5432/tracker?sslmode=disable
      JWT_SECRET: dev-secret-change-me
      NTFY_URL: http://ntfy:80
      PORT: "8080"
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  pgdata:
  ntfy-cache:
```

- [ ] **Step 3: Update .env.example**

Replace `FCM_CREDENTIALS_FILE=` with:

```
NTFY_URL=http://localhost:8090
```

- [ ] **Step 4: Remove Firebase dependencies from go.mod**

```bash
cd server && go mod tidy
```

This will remove unused `firebase.google.com/go/v4` and its transitive deps.

- [ ] **Step 5: Verify build**

```bash
cd server && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: wire ntfy into main.go and Docker Compose, remove Firebase deps"
```

---

## Task 4: Android — Remove Firebase, Add ntfy SSE Listener

**Files:**
- Delete: `android/app/src/main/java/com/nschatz/tracker/fcm/TrackerFirebaseService.kt`
- Delete: `android/app/src/main/java/com/nschatz/tracker/data/api/FcmApi.kt`
- Modify: `android/app/build.gradle.kts`
- Modify: `android/app/src/main/AndroidManifest.xml`
- Modify: `android/app/src/main/java/com/nschatz/tracker/data/api/ApiClient.kt`
- Modify: `android/app/src/main/java/com/nschatz/tracker/data/model/ApiModels.kt`
- Modify: `android/app/src/main/java/com/nschatz/tracker/data/prefs/SessionManager.kt`
- Modify: `android/app/src/main/java/com/nschatz/tracker/service/LocationService.kt`
- Modify: `android/app/src/main/java/com/nschatz/tracker/ui/main/MainActivity.kt`
- Modify: `android/build.gradle.kts`

- [ ] **Step 1: Delete Firebase files**

```bash
rm android/app/src/main/java/com/nschatz/tracker/fcm/TrackerFirebaseService.kt
rm android/app/src/main/java/com/nschatz/tracker/data/api/FcmApi.kt
```

- [ ] **Step 2: Remove Firebase from build.gradle.kts files**

In `android/build.gradle.kts` (project-level), remove:
```kotlin
id("com.google.gms.google-services") version "4.4.2" apply false
```

In `android/app/build.gradle.kts`, remove:
```kotlin
id("com.google.gms.google-services")
```

And remove these dependencies:
```kotlin
implementation(platform("com.google.firebase:firebase-bom:33.1.2"))
implementation("com.google.firebase:firebase-messaging")
```

- [ ] **Step 3: Remove Firebase service from AndroidManifest.xml**

Remove this entire block:
```xml
        <service
            android:name=".fcm.TrackerFirebaseService"
            android:exported="false">
            <intent-filter>
                <action android:name="com.google.firebase.MESSAGING_EVENT" />
            </intent-filter>
        </service>
```

- [ ] **Step 4: Remove FcmApi from ApiClient.kt**

Remove the line:
```kotlin
val fcm: FcmApi by lazy { retrofit.create(FcmApi::class.java) }
```

- [ ] **Step 5: Remove FcmTokenRequest from ApiModels.kt**

Remove:
```kotlin
data class FcmTokenRequest(val token: String)
```

- [ ] **Step 6: Add ntfyUrl to SessionManager.kt**

Add property:
```kotlin
var ntfyUrl: String
    get() {
        val custom = prefs.getString("ntfy_url", null)
        if (custom != null) return custom
        // Derive from server URL: replace port with 8090
        val base = serverUrl.replace(Regex(":\\d+$"), "")
        return "$base:8090"
    }
    set(value) = prefs.edit().putString("ntfy_url", value).apply()
```

Add computed property:
```kotlin
val ntfyTopic: String get() = "tracker-$userId"
```

- [ ] **Step 7: Add ntfy SSE listener to LocationService.kt**

Add these fields to LocationService:

```kotlin
private var sseCall: okhttp3.Call? = null
private val sseClient = okhttp3.OkHttpClient.Builder()
    .readTimeout(0, java.util.concurrent.TimeUnit.SECONDS)
    .build()
```

Add method to start SSE subscription:

```kotlin
private fun startNtfyListener() {
    val userId = sessionManager.userId ?: return
    val ntfyUrl = sessionManager.ntfyUrl.trimEnd('/')
    val topic = "tracker-$userId"
    val url = "$ntfyUrl/$topic/sse"

    val request = okhttp3.Request.Builder()
        .url(url)
        .header("Accept", "text/event-stream")
        .build()

    sseCall = sseClient.newCall(request)
    serviceScope.launch(Dispatchers.IO) {
        try {
            val response = sseCall?.execute() ?: return@launch
            val source = response.body?.source() ?: return@launch
            val gson = com.google.gson.Gson()

            while (!source.exhausted()) {
                val line = source.readUtf8Line() ?: continue
                if (!line.startsWith("data: ")) continue

                try {
                    val json = line.removePrefix("data: ")
                    val event = gson.fromJson(json, NtfyEvent::class.java)
                    if (event.event == "message") {
                        showNotification(event.title ?: "Tracker", event.message ?: "")
                    }
                } catch (e: Exception) {
                    // Skip malformed lines
                }
            }
        } catch (e: java.io.IOException) {
            // Connection lost — reconnect after delay
            if (sseCall?.isCanceled() != true) {
                kotlinx.coroutines.delay(5000)
                startNtfyListener()
            }
        }
    }
}

private fun showNotification(title: String, body: String) {
    val intent = android.app.PendingIntent.getActivity(
        this, 0,
        Intent(this, com.nschatz.tracker.ui.main.MainActivity::class.java),
        android.app.PendingIntent.FLAG_IMMUTABLE
    )

    val notification = NotificationCompat.Builder(this, TrackerApp.CHANNEL_PLACE_ALERTS)
        .setSmallIcon(android.R.drawable.ic_dialog_map)
        .setContentTitle(title)
        .setContentText(body)
        .setContentIntent(intent)
        .setAutoCancel(true)
        .setPriority(NotificationCompat.PRIORITY_HIGH)
        .build()

    val manager = getSystemService(NotificationManager::class.java)
    manager.notify(System.currentTimeMillis().toInt(), notification)
}

private data class NtfyEvent(
    val event: String? = null,
    val title: String? = null,
    val message: String? = null
)
```

Call `startNtfyListener()` in the `ACTION_START` branch of `onStartCommand`, after `registerActivityTransitions()`.

In `onStartCommand` ACTION_STOP branch, add `sseCall?.cancel()` before stopping.

- [ ] **Step 8: Remove FCM token registration from MainActivity.kt**

Remove the `registerFcmToken()` method entirely and its call in `onCreate`. Also remove:
```kotlin
import com.google.firebase.messaging.FirebaseMessaging
import com.nschatz.tracker.data.model.FcmTokenRequest
```

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "feat: remove Firebase from Android, add ntfy SSE listener in LocationService"
```

---

## Summary

| Task | Scope | What changes |
|------|-------|-------------|
| 1 | Go notify package | Replace fcm.go with ntfy.go, rename params to userIDs |
| 2 | Go API + store | Remove FCMTokenStore, /fcm-token route, update NewServer to 9 params |
| 3 | Go main + infra | Wire ntfy in main.go, add to Docker Compose, remove Firebase deps |
| 4 | Android | Remove Firebase entirely, add ntfy SSE listener in LocationService |
