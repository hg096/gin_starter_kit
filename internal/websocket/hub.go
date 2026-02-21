package websocket

import (
	"gin_starter/pkg/logger"
	"sync"
)

// Hub manages websocket clients and room broadcasts.
type Hub struct {
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// Message is the websocket payload.
type Message struct {
	Type    string      `json:"type"`
	Room    string      `json:"room"`
	UserID  string      `json:"user_id"`
	Content interface{} `json:"content"`
}

// NewHub creates a websocket hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Publish broadcasts a message to connected clients.
func (h *Hub) Publish(message *Message) {
	if message == nil {
		return
	}

	select {
	case h.broadcast <- message:
	default:
		logger.Warn("websocket broadcast channel is full; message dropped type=%s room=%s", message.Type, message.Room)
	}
}

// Run starts the websocket event loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true

	if client.RoomID != "" {
		if h.rooms[client.RoomID] == nil {
			h.rooms[client.RoomID] = make(map[*Client]bool)
		}
		h.rooms[client.RoomID][client] = true

		logger.Info("client connected: %s (room %s)", client.UserID, client.RoomID)

		h.Publish(&Message{
			Type:   "join",
			Room:   client.RoomID,
			UserID: client.UserID,
			Content: map[string]interface{}{
				"message": client.UserID + "님이 입장했습니다",
			},
		})
	}
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)

		if client.RoomID != "" {
			if clients, exists := h.rooms[client.RoomID]; exists {
				delete(clients, client)

				if len(clients) == 0 {
					delete(h.rooms, client.RoomID)
					logger.Info("room removed: %s", client.RoomID)
				}
			}

			logger.Info("client disconnected: %s (room %s)", client.UserID, client.RoomID)

			h.Publish(&Message{
				Type:   "leave",
				Room:   client.RoomID,
				UserID: client.UserID,
				Content: map[string]interface{}{
					"message": client.UserID + "님이 퇴장했습니다",
				},
			})
		}

		close(client.send)
	}
}

func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if message.Room != "" {
		if clients, ok := h.rooms[message.Room]; ok {
			for client := range clients {
				select {
				case client.send <- message:
				default:
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
		}
		return
	}

	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}
}

// GetRoomClients returns connected user IDs for a room.
func (h *Hub) GetRoomClients(roomID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userIDs := make([]string, 0)
	if clients, ok := h.rooms[roomID]; ok {
		for client := range clients {
			userIDs = append(userIDs, client.UserID)
		}
	}
	return userIDs
}

// GetRoomCount returns number of active rooms.
func (h *Hub) GetRoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}

// GetClientCount returns number of active clients.
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
