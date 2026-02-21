package websocket

import (
	"gin_starter/pkg/logger"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

// Client represents one websocket client connection.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan *Message
	UserID string
	RoomID string
}

// NewClient creates a websocket client wrapper.
func NewClient(hub *Hub, conn *websocket.Conn, userID, roomID string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan *Message, 256),
		UserID: userID,
		RoomID: roomID,
	}
}

// ReadPump reads client messages and forwards them to hub.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
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
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
			) {
				logger.Warn("websocket read error: %v", err)
			}
			break
		}

		message.UserID = c.UserID
		message.Room = c.RoomID
		c.hub.Publish(&message)
	}
}

// WritePump writes outbound messages and keepalive pings.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				logger.Error("websocket write error: %v", err)
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
