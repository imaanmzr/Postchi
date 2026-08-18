package collab

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/imaanmzr/postchi/backend/internal/auth"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	mu      sync.RWMutex
	rooms   map[string]map[*Client]struct{}
	log     *zap.Logger
}

type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	workspaceID string
	userID      string
	email       string
}

type Message struct {
	Type        string `json:"type"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	Email       string `json:"email,omitempty"`
	EntityType  string `json:"entity_type,omitempty"`
	EntityID    string `json:"entity_id,omitempty"`
	Payload     any    `json:"payload,omitempty"`
}

func NewHub(log *zap.Logger) *Hub {
	return &Hub{rooms: map[string]map[*Client]struct{}{}, log: log}
}

func (h *Hub) Broadcast(workspaceID string, msg any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room := h.rooms[workspaceID]
	data, _ := json.Marshal(msg)
	for c := range room {
		_ = c.conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.UserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	wsID := chi.URLParam(r, "workspaceId")
	if wsID == "" {
		wsID = r.URL.Query().Get("workspace_id")
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &Client{hub: h, conn: conn, workspaceID: wsID, userID: userID.String()}
	h.register(client)
	defer h.unregister(client)

	presence := Message{Type: "presence.join", WorkspaceID: wsID, UserID: client.userID}
	h.Broadcast(wsID, presence)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var msg Message
		if json.Unmarshal(data, &msg) == nil {
			msg.WorkspaceID = wsID
			msg.UserID = client.userID
			h.Broadcast(wsID, msg)
		}
	}
	h.Broadcast(wsID, Message{Type: "presence.leave", WorkspaceID: wsID, UserID: client.userID})
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[c.workspaceID] == nil {
		h.rooms[c.workspaceID] = map[*Client]struct{}{}
	}
	h.rooms[c.workspaceID][c] = struct{}{}
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms[c.workspaceID], c)
	_ = c.conn.Close()
}
