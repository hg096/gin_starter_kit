package websocket

import (
	"gin_starter/pkg/logger"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 클라이언트로 메시지 쓰기 대기 시간
	writeWait = 10 * time.Second

	// 클라이언트로부터 다음 pong 메시지 대기 시간
	pongWait = 60 * time.Second

	// ping 전송 주기 (pongWait보다 작아야 함)
	pingPeriod = (pongWait * 9) / 10

	// 최대 메시지 크기
	maxMessageSize = 512 * 1024 // 512KB
)

// Client WebSocket 클라이언트
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan *Message
	UserID string
	RoomID string
}

// NewClient 클라이언트 생성
func NewClient(hub *Hub, conn *websocket.Conn, userID, roomID string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan *Message, 256),
		UserID: userID,
		RoomID: roomID,
	}
}

// ReadPump 클라이언트로부터 메시지 읽기
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	c.conn.SetReadLimit(maxMessageSize)

	for {
		var message Message
		err := c.conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket 읽기 오류: %v", err)
			}
			break
		}

		// 메시지에 사용자 정보 추가
		message.UserID = c.UserID
		message.Room = c.RoomID

		// 메시지 브로드캐스트
		c.hub.broadcast <- &message
	}
}

// WritePump 클라이언트에게 메시지 쓰기
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub가 채널을 닫음
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// JSON 메시지 전송
			if err := c.conn.WriteJSON(message); err != nil {
				logger.Error("WebSocket 쓰기 오류: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}