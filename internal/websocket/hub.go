package websocket

import (
	"gin_starter/pkg/logger"
	"sync"
)

// Hub WebSocket 연결 관리
type Hub struct {
	clients    map[*Client]bool      // 연결된 클라이언트들
	rooms      map[string]map[*Client]bool // 방별 클라이언트
	broadcast  chan *Message         // 브로드캐스트 메시지
	register   chan *Client          // 클라이언트 등록
	unregister chan *Client          // 클라이언트 해제
	mu         sync.RWMutex          // 동시성 제어
}

// Message WebSocket 메시지 구조
type Message struct {
	Type    string      `json:"type"`    // message, join, leave, etc.
	Room    string      `json:"room"`    // 방 ID
	UserID  string      `json:"user_id"` // 사용자 ID
	Content interface{} `json:"content"` // 메시지 내용
}

// NewHub Hub 생성
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run Hub 실행 (고루틴으로 실행)
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

// registerClient 클라이언트 등록
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true

	// 방에 클라이언트 추가
	if client.RoomID != "" {
		if h.rooms[client.RoomID] == nil {
			h.rooms[client.RoomID] = make(map[*Client]bool)
		}
		h.rooms[client.RoomID][client] = true

		logger.Info("클라이언트 등록: %s (방: %s)", client.UserID, client.RoomID)

		// 입장 메시지 브로드캐스트
		h.broadcast <- &Message{
			Type:   "join",
			Room:   client.RoomID,
			UserID: client.UserID,
			Content: map[string]interface{}{
				"message": client.UserID + "님이 입장했습니다",
			},
		}
	}
}

// unregisterClient 클라이언트 해제
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)

		// 방에서 클라이언트 제거
		if client.RoomID != "" {
			if clients, ok := h.rooms[client.RoomID]; ok {
				delete(clients, client)

				// 방에 클라이언트가 없으면 방 삭제
				if len(clients) == 0 {
					delete(h.rooms, client.RoomID)
					logger.Info("방 삭제: %s", client.RoomID)
				}
			}

			logger.Info("클라이언트 해제: %s (방: %s)", client.UserID, client.RoomID)

			// 퇴장 메시지 브로드캐스트
			h.broadcast <- &Message{
				Type:   "leave",
				Room:   client.RoomID,
				UserID: client.UserID,
				Content: map[string]interface{}{
					"message": client.UserID + "님이 퇴장했습니다",
				},
			}
		}

		close(client.send)
	}
}

// broadcastMessage 메시지 브로드캐스트
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 특정 방에만 전송
	if message.Room != "" {
		if clients, ok := h.rooms[message.Room]; ok {
			for client := range clients {
				select {
				case client.send <- message:
				default:
					// 전송 실패 시 클라이언트 제거
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
		}
	} else {
		// 모든 클라이언트에게 전송
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
}

// GetRoomClients 방의 클라이언트 목록 조회
func (h *Hub) GetRoomClients(roomID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var userIDs []string
	if clients, ok := h.rooms[roomID]; ok {
		for client := range clients {
			userIDs = append(userIDs, client.UserID)
		}
	}

	return userIDs
}

// GetRoomCount 방 개수 조회
func (h *Hub) GetRoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.rooms)
}

// GetClientCount 전체 클라이언트 수 조회
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}