package routes

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type chatMessage struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Content string `json:"content"`
	Time    int64  `json:"time"`
}

type chatClient struct {
	id   string
	conn *websocket.Conn
	hub  *chatHub
	send chan chatMessage
}

type chatHub struct {
	mu      sync.Mutex
	clients map[string]map[*chatClient]bool
	history map[string][]chatMessage
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var hub = newChatHub()

func newChatHub() *chatHub {
	h := &chatHub{
		clients: make(map[string]map[*chatClient]bool),
		history: make(map[string][]chatMessage),
	}
	return h
}

func (h *chatHub) register(c *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.id] == nil {
		h.clients[c.id] = make(map[*chatClient]bool)
	}
	h.clients[c.id][c] = true
	// send history
	for _, m := range h.history[c.id] {
		c.send <- m
	}
}

func (h *chatHub) unregister(c *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if cl, ok := h.clients[c.id]; ok {
		if _, ok := cl[c]; ok {
			delete(cl, c)
			close(c.send)
		}
		if len(cl) == 0 {
			delete(h.clients, c.id)
		}
	}
}

func (h *chatHub) broadcast(msg chatMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.history[msg.To] = append(h.history[msg.To], msg)
	if cl, ok := h.clients[msg.To]; ok {
		for c := range cl {
			select {
			case c.send <- msg:
			default:
				close(c.send)
				delete(cl, c)
			}
		}
	}
}

func (c *chatClient) readPump(target string) {
	defer func() {
		c.hub.unregister(c)
		c.conn.Close()
	}()
	for {
		var msg chatMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg.To == "" {
			msg.To = target
		}
		c.hub.broadcast(msg)
	}
}

func (c *chatClient) writePump() {
	for msg := range c.send {
		c.conn.WriteJSON(msg)
	}
}

// ChatWebSocket handles /ws endpoint
func ChatWebSocket(c *gin.Context) {
	userID := c.Query("user")
	target := c.Query("target")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &chatClient{id: userID, conn: conn, hub: hub, send: make(chan chatMessage, 8)}
	hub.register(client)
	go client.writePump()
	client.readPump(target)
}

func SetupChatRoutes(r *gin.Engine) {
	r.GET("/ws", ChatWebSocket)
}
