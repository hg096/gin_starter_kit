package routes

import (
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

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
	mu      sync.RWMutex
	clients map[string]map[*chatClient]bool
	// history map[string][]chatMessage
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var hub = newChatHub()

func newChatHub() *chatHub {
	return &chatHub{
		clients: make(map[string]map[*chatClient]bool),
		// history: make(map[string][]chatMessage),
	}
}

func (h *chatHub) register(c *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[c.room] == nil {
		h.clients[c.room] = make(map[*chatClient]bool)
	}
	h.clients[c.room][c] = true

	// send existing chat history (non-blocking)
	// go func(msgs []chatMessage) {
	// 	for _, m := range msgs {
	// 		select {
	// 		case c.send <- m:
	// 		case <-time.After(100 * time.Millisecond):
	// 			log.Println("[register] Timeout sending history to", c.id)
	// 		}
	// 	}
	// }(h.history[c.room])
}

func (h *chatHub) unregister(c *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[c.room]; ok {
		if _, exists := clients[c]; exists {
			delete(clients, c)
			close(c.send)
		}
		if len(clients) == 0 {
			delete(h.clients, c.room)
			// delete(h.history, c.room)
		}
	}
}

func (h *chatHub) broadcast(room string, msg chatMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// h.history[room] = append(h.history[room], msg)
	if clients, ok := h.clients[room]; ok {
		for c := range clients {
			select {
			case c.send <- msg:
			default:
				log.Println("[broadcast] client send buffer full, disconnecting:", c.id)
				close(c.send)
				delete(clients, c)
			}
		}
	}
}

func (c *chatClient) readPump() {
	defer func() {
		log.Println("[readPump] closing:", c.id)
		c.hub.unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	for {
		var msg chatMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			log.Println("[readPump] ReadJSON error:", err)
			break
		}
		c.hub.broadcast(c.room, msg)
	}
}

func (c *chatClient) writePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				log.Println("[writePump] WriteJSON error:", err)
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("[writePump] Ping error:", err)
				return
			}
		}
	}
}

func ChatWebSocket(c *gin.Context) {
	userID := c.Query("user")
	target := c.Query("target")
	room := makeRoomID(userID, target)

	// 커넥션 로그
	// log.Println("[WebSocket] new connection:", userID, "→", target, "Room:", room)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("[WebSocket] upgrade error:", err)
		return
	}

	client := &chatClient{
		id:   userID,
		room: room,
		conn: conn,
		hub:  hub,
		send: make(chan chatMessage, 16),
	}

	hub.register(client)
	go client.writePump()
	client.readPump()
}

func SetupChatRoutes(r *gin.Engine) {
	r.GET("/ws", ChatWebSocket)
}
