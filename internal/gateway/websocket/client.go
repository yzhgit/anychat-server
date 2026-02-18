package websocket

import (
	"encoding/json"
	"time"

	"github.com/anychat/server/pkg/logger"
	gorillaws "github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second // 写操作超时
	pongWait       = 60 * time.Second // 等待pong响应的超时
	pingPeriod     = 54 * time.Second // 发送ping的间隔（小于pongWait）
	maxMessageSize = 65536            // 最大消息大小（64KB）
)

// Message WebSocket消息格式
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Client WebSocket客户端
type Client struct {
	UserID   string
	DeviceID string
	Conn     *gorillaws.Conn
	Send     chan []byte    // 待发送消息队列
	Done     chan struct{}  // 关闭信号（被新连接替换时关闭）
	manager  *Manager
}

// NewClient 创建新的WebSocket客户端
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

// WritePump 处理向客户端发送消息（含心跳），在独立goroutine中运行
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
				// Send通道已关闭
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
			// 被新连接替换，关闭旧连接
			c.Conn.WriteMessage(gorillaws.CloseMessage, gorillaws.FormatCloseMessage(
				gorillaws.CloseNormalClosure, "replaced by new connection"))
			return
		}
	}
}

// ReadPump 处理从客户端接收消息，阻塞直到连接断开
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
