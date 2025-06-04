package routes

import (
	"log"
	"net/http"
	"sort"
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

func makeRoomID(a, b string) string {
	pair := []string{a, b}
	sort.Strings(pair) // 항상 같은 순서
	return pair[0] + ":" + pair[1]
}

type chatClient struct {
	id   string
	room string
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
	if h.clients[c.room] == nil {
		h.clients[c.room] = make(map[*chatClient]bool)
	}
	h.clients[c.room][c] = true
	// send history
	for _, m := range h.history[c.room] {
		c.send <- m
	}
}

func (h *chatHub) unregister(c *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if cl, ok := h.clients[c.room]; ok {
		if _, ok := cl[c]; ok {
			delete(cl, c)
			close(c.send)
		}
		// ✅ 참가자 남아있으면 유지
		if len(cl) == 0 {
			delete(h.clients, c.room)
			delete(h.history, c.room)
		}
	}
}

func (h *chatHub) broadcast(room string, msg chatMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.history[room] = append(h.history[room], msg)
	if cl, ok := h.clients[room]; ok {
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

func (c *chatClient) readPump() {
	defer func() {
		c.hub.unregister(c)
		c.conn.Close()
	}()
	for {
		var msg chatMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			break
		}
		c.hub.broadcast(c.room, msg)
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
	room := makeRoomID(userID, target)

	log.Println("New WebSocket:", userID, "→", target, "Room:", room)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &chatClient{id: userID, room: room, conn: conn, hub: hub, send: make(chan chatMessage, 8)}
	hub.register(client)
	go client.writePump()
	client.readPump()
}

func SetupChatRoutes(r *gin.Engine) {
	r.GET("/ws", ChatWebSocket)
}
