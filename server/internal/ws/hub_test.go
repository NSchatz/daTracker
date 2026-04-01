package ws_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nschatz/tracker/server/internal/model"
	"github.com/nschatz/tracker/server/internal/ws"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func TestHubBroadcast(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	userID := uuid.New()
	circleID := uuid.New()

	// Create httptest.Server that calls hub.HandleConnect for a specific userID/circleID
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleConnect(w, r, userID, circleID)
	}))
	defer srv.Close()

	// Connect via websocket (nhooyr.io/websocket client)
	wsURL := "ws" + srv.URL[4:] // replace "http" with "ws"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Wait briefly for registration
	time.Sleep(50 * time.Millisecond)

	// BroadcastLocation for the same circleID
	loc := model.Location{
		ID:         1,
		UserID:     userID,
		Lat:        37.7749,
		Lng:        -122.4194,
		RecordedAt: time.Now().UTC().Truncate(time.Second),
	}
	hub.BroadcastLocation(circleID, loc)

	// Read the message from the websocket connection
	var received model.Location
	if err := wsjson.Read(ctx, conn, &received); err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	// Verify it's the correct location JSON
	if received.UserID != loc.UserID {
		t.Errorf("expected userID %s, got %s", loc.UserID, received.UserID)
	}
	if received.Lat != loc.Lat {
		t.Errorf("expected lat %f, got %f", loc.Lat, received.Lat)
	}
	if received.Lng != loc.Lng {
		t.Errorf("expected lng %f, got %f", loc.Lng, received.Lng)
	}

	// Verify IsConnected returns true for the connected user
	if !hub.IsConnected(userID) {
		t.Error("expected user to be connected")
	}

	// Verify IsConnected returns false for an unknown user
	if hub.IsConnected(uuid.New()) {
		t.Error("expected unknown user to not be connected")
	}
}
