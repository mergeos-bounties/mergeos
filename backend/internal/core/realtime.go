package core

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	realtimeWriteWait      = 10 * time.Second
	realtimePongWait      = 60 * time.Second
	realtimePingPeriod     = (realtimePongWait * 9) / 10
	realtimeMessageLimit   = 1 << 20
	realtimeSendBufferSize = 16
)

var realtimeUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

type RealtimeHub struct {
	mu        sync.RWMutex
	public    map[*realtimeClient]struct{}
	dashboard map[*realtimeClient]struct{}
}

func NewRealtimeHub() *RealtimeHub {
	return &RealtimeHub{
		public:    map[*realtimeClient]struct{}{},
		dashboard: map[*realtimeClient]struct{}{},
	}
}

type realtimeClient struct {
	hub       *RealtimeHub
	conn      *websocket.Conn
	send      chan []byte
	scope     string
	userID    string
	isAdmin   bool
	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
}

type publicProjectRealtimeEvent struct {
	Type       string             `json:"type"`
	Marketplace MarketplaceResponse `json:"marketplace"`
}

type dashboardProjectRealtimeEvent struct {
	Type    string   `json:"type"`
	Project *Project `json:"project"`
}

func (s *Server) publicWebSocket(w http.ResponseWriter, r *http.Request) {
	s.realtime.servePublic(w, r)
}

func (s *Server) dashboardWebSocket(w http.ResponseWriter, r *http.Request) {
	userToken := strings.TrimSpace(r.URL.Query().Get("token"))
	if userToken == "" {
		writeError(w, http.StatusUnauthorized, "login is required")
		return
	}
	user, ok := s.store.UserByToken(userToken)
	if !ok {
		writeError(w, http.StatusUnauthorized, "login is required")
		return
	}
	s.realtime.serveDashboard(w, r, user)
}

func (s *Server) publishProjectEvent(project *Project) {
	if s.realtime == nil || project == nil {
		return
	}
	s.realtime.broadcastPublic(publicProjectRealtimeEvent{
		Type:       "project-funded",
		Marketplace: s.store.Marketplace(),
	})
	s.realtime.broadcastDashboard(project)
}

func (h *RealtimeHub) servePublic(w http.ResponseWriter, r *http.Request) {
	conn, err := realtimeUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &realtimeClient{
		hub:   h,
		conn:  conn,
		send:  make(chan []byte, realtimeSendBufferSize),
		scope: "public",
	}
	h.register(client)
	go client.writePump()
	client.readPump()
}

func (h *RealtimeHub) serveDashboard(w http.ResponseWriter, r *http.Request, user *User) {
	conn, err := realtimeUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &realtimeClient{
		hub:     h,
		conn:    conn,
		send:    make(chan []byte, realtimeSendBufferSize),
		scope:   "dashboard",
		userID:  user.ID,
		isAdmin: normalizeRole(user.Role) == RoleAdmin,
	}
	h.register(client)
	go client.writePump()
	client.readPump()
}

func (h *RealtimeHub) broadcastPublic(event publicProjectRealtimeEvent) {
	message, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	clients := make([]*realtimeClient, 0, len(h.public))
	for client := range h.public {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	for _, client := range clients {
		if !client.enqueue(message) {
			client.close()
		}
	}
}

func (h *RealtimeHub) broadcastDashboard(project *Project) {
	if project == nil {
		return
	}
	event := dashboardProjectRealtimeEvent{Type: "project-funded", Project: cloneProject(project)}
	message, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	clients := make([]*realtimeClient, 0, len(h.dashboard))
	for client := range h.dashboard {
		if client.isAdmin || client.userID == project.ClientUserID {
			clients = append(clients, client)
		}
	}
	h.mu.RUnlock()
	for _, client := range clients {
		if !client.enqueue(message) {
			client.close()
		}
	}
}

func (h *RealtimeHub) register(client *realtimeClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if client.scope == "dashboard" {
		h.dashboard[client] = struct{}{}
		return
	}
	h.public[client] = struct{}{}
}

func (h *RealtimeHub) unregister(client *realtimeClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if client.scope == "dashboard" {
		delete(h.dashboard, client)
		return
	}
	delete(h.public, client)
}

func (c *realtimeClient) enqueue(message []byte) bool {
	copyMessage := append([]byte(nil), message...)
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return false
	}
	select {
	case c.send <- copyMessage:
		return true
	default:
		return false
	}
}

func (c *realtimeClient) close() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		close(c.send)
		c.mu.Unlock()
		c.hub.unregister(c)
		_ = c.conn.Close()
	})
}

func (c *realtimeClient) readPump() {
	defer c.close()
	c.conn.SetReadLimit(realtimeMessageLimit)
	_ = c.conn.SetReadDeadline(time.Now().Add(realtimePongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(realtimePongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *realtimeClient) writePump() {
	ticker := time.NewTicker(realtimePingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(realtimeWriteWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(realtimeWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}