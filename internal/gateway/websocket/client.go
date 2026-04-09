package websocket

import (
	"encoding/json"
	"time"

	"github.com/anychat/server/pkg/logger"
	gorillaws "github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second // write operation timeout
	pongWait       = 60 * time.Second // timeout waiting for pong response
	pingPeriod     = 54 * time.Second // interval to send ping (less than pongWait)
	maxMessageSize = 65536            // max message size (64KB)
)

// Message WebSocket message format
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Client WebSocket client
type Client struct {
	UserID   string
	DeviceID string
	Conn     *gorillaws.Conn
	Send     chan []byte   // message queue to send
	Done     chan struct{} // close signal (closed when replaced by new connection)
	manager  *Manager
}

// NewClient creates new WebSocket client
func NewClient(userID, deviceID string, conn *gorillaws.Conn, manager *Manager) *Client {
	return &Client{
		UserID:   userID,
		DeviceID: deviceID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Done:     make(chan struct{}),
		manager:  manager,
	}
}

// WritePump handles sending messages to client (with heartbeat), runs in separate goroutine
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Send channel is closed
				c.Conn.WriteMessage(gorillaws.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(gorillaws.TextMessage, message); err != nil {
				logger.Warn("WebSocket write error",
					zap.String("userID", c.UserID),
					zap.Error(err))
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(gorillaws.PingMessage, nil); err != nil {
				return
			}
		case <-c.Done:
			// replaced by new connection, close old connection
			c.Conn.WriteMessage(gorillaws.CloseMessage, gorillaws.FormatCloseMessage(
				gorillaws.CloseNormalClosure, "replaced by new connection"))
			return
		}
	}
}

// ReadPump handles receiving messages from client, blocks until connection disconnects
func (c *Client) ReadPump(onMessage func(client *Client, msg *Message)) {
	defer func() {
		c.manager.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			if gorillaws.IsUnexpectedCloseError(err, gorillaws.CloseGoingAway, gorillaws.CloseAbnormalClosure) {
				logger.Warn("WebSocket unexpected close",
					zap.String("userID", c.UserID),
					zap.Error(err))
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			logger.Warn("Failed to parse WebSocket message",
				zap.String("userID", c.UserID),
				zap.Error(err))
			continue
		}

		onMessage(c, &msg)
	}
}
