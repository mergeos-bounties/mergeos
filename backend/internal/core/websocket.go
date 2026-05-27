package core

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Hub broadcasts project events to all connected WebSocket clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

// ProjectEvent is emitted when a project is created or funded.
type ProjectEvent struct {
	Type    string `json:"type"`    // "project.created", "project.funded"
	Project any    `json:"project"` // Project summary
	Version int    `json:"version"`
}

var globalHub = &Hub{clients: make(map[*websocket.Conn]bool)}

func init() {
	go globalHub.periodicCleanup()
}

// broadcast sends an event to all connected clients.
func (h *Hub) broadcast(event ProjectEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[ws] marshal error: %v", err)
		return
	}
	for client := range h.clients {
		select {
		case client.WriteMessage(websocket.TextMessage, data):
		default:
			log.Printf("[ws] dropping slow client")
		}
	}
}

// register adds a client.
func (h *Hub) register(client *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
}

// unregister removes a client.
func (h *Hub) unregister(client *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client)
	client.Close()
}

// periodicCleanup removes stale connections every 5 minutes.
func (h *Hub) periodicCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		h.mu.Lock()
		for client := range h.clients {
			if err := client.WriteMessage(websocket.PingMessage, nil); err != nil {
				delete(h.clients, client)
				client.Close()
			}
		}
		h.mu.Unlock()
	}
}

// wsHandler upgrades HTTP to WebSocket and registers the client.
func (s *Server) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}
	globalHub.register(conn)
	log.Printf("[ws] client connected (%d total)", len(globalHub.clients))

	// Read loop (keeps connection alive; we only send events, so discard reads)
	go func() {
		defer globalHub.unregister(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// BroadcastProjectEvent publishes a project event to all WS clients.
// Call this from createProject / payment handler after project creation.
func BroadcastProjectEvent(evt ProjectEvent) {
	globalHub.broadcast(evt)
}
