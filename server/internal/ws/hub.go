package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

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
		register:   make(chan *client, 16),
		unregister: make(chan *client, 16),
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
		log.Printf("ws: accept error: %v", err)
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

	// Read loop: keeps connection alive and detects close
	defer func() {
		h.unregister <- c
		cancel()
		conn.Close(websocket.StatusNormalClosure, "")
	}()

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
		log.Printf("ws: marshal error: %v", err)
		return
	}

	h.mu.RLock()
	targets := make([]*client, 0)
	for c := range h.clients {
		if c.circleID == circleID {
			targets = append(targets, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range targets {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := c.conn.Write(ctx, websocket.MessageText, data)
		cancel()
		if err != nil {
			log.Printf("ws: write error to client %s: %v", c.userID, err)
		}
	}
}

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
