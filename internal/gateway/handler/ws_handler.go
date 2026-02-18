package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	messagepb "github.com/anychat/server/api/proto/message"
	"github.com/anychat/server/internal/gateway/client"
	gwnotification "github.com/anychat/server/internal/gateway/notification"
	"github.com/anychat/server/internal/gateway/websocket"
	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/logger"
	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有来源（生产环境应验证Origin）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WSHandler WebSocket处理器
type WSHandler struct {
	clientManager *client.Manager
	jwtManager    *jwt.Manager
	wsManager     *websocket.Manager
	subscriber    *gwnotification.Subscriber
}

// NewWSHandler 创建WebSocket处理器
func NewWSHandler(
	clientManager *client.Manager,
	jwtManager *jwt.Manager,
	wsManager *websocket.Manager,
	subscriber *gwnotification.Subscriber,
) *WSHandler {
	return &WSHandler{
		clientManager: clientManager,
		jwtManager:    jwtManager,
		wsManager:     wsManager,
		subscriber:    subscriber,
	}
}

// HandleWebSocket 处理WebSocket连接
// @Summary      建立WebSocket长连接
// @Description  客户端通过WebSocket保持长连接，接收实时通知和消息推送。由于WebSocket协议不支持自定义Header，JWT token通过URL query参数传入。
// @Tags         实时通信
// @Param        token  query  string  true  "JWT access token"
// @Router       /ws [get]
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token is required"})
		return
	}

	claims, err := h.jwtManager.ValidateAccessToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	userID := claims.UserID
	deviceID := claims.DeviceID

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("Failed to upgrade WebSocket connection", zap.Error(err))
		return
	}

	wsClient := websocket.NewClient(userID, deviceID, conn, h.wsManager)
	h.wsManager.Register(wsClient)

	if err := h.subscriber.SubscribeUser(userID); err != nil {
		logger.Error("Failed to subscribe user notifications",
			zap.String("userID", userID),
			zap.Error(err))
	}

	logger.Info("WebSocket client connected",
		zap.String("userID", userID),
		zap.String("deviceID", deviceID),
		zap.Int("onlineCount", h.wsManager.OnlineCount()))

	go wsClient.WritePump()

	// ReadPump阻塞直到连接断开
	wsClient.ReadPump(h.handleClientMessage)

	// 连接断开后，仅在用户真正离线时才取消NATS订阅
	// IsOnline返回false表示该用户没有新的活跃连接（未被替换）
	if !h.wsManager.IsOnline(userID) {
		h.subscriber.UnsubscribeUser(userID)
	}

	logger.Info("WebSocket client disconnected",
		zap.String("userID", userID),
		zap.Int("onlineCount", h.wsManager.OnlineCount()))
}

// handleClientMessage 处理客户端发来的WebSocket消息
func (h *WSHandler) handleClientMessage(c *websocket.Client, msg *websocket.Message) {
	switch msg.Type {
	case "ping":
		pong := &websocket.Message{Type: "pong"}
		h.wsManager.SendMessageToUser(c.UserID, pong)

	case "message.send":
		h.handleSendMessage(c, msg.Payload)

	default:
		logger.Debug("Unknown WebSocket message type",
			zap.String("type", msg.Type),
			zap.String("userID", c.UserID))
	}
}

// sendMessagePayload 客户端发送消息的payload结构
type sendMessagePayload struct {
	ConversationID   string   `json:"conversationId"`
	ConversationType string   `json:"conversationType"`
	ContentType      string   `json:"contentType"`
	Content          string   `json:"content"`
	ReplyTo          string   `json:"replyTo,omitempty"`
	AtUsers          []string `json:"atUsers,omitempty"`
	LocalID          string   `json:"localId,omitempty"`
}

// sendMessageResult 发送消息响应结构
type sendMessageResult struct {
	MessageID string `json:"messageId"`
	Sequence  int64  `json:"sequence"`
	Timestamp int64  `json:"timestamp"`
	LocalID   string `json:"localId,omitempty"`
}

// handleSendMessage 解析message.send payload并通过gRPC转发
func (h *WSHandler) handleSendMessage(c *websocket.Client, payload json.RawMessage) {
	var req sendMessagePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		logger.Warn("Invalid message.send payload",
			zap.String("userID", c.UserID),
			zap.Error(err))
		return
	}

	grpcReq := &messagepb.SendMessageRequest{
		SenderId:         c.UserID,
		ConversationId:   req.ConversationID,
		ConversationType: req.ConversationType,
		ContentType:      req.ContentType,
		Content:          req.Content,
	}
	if req.LocalID != "" {
		grpcReq.LocalId = &req.LocalID
	}
	if req.ReplyTo != "" {
		grpcReq.ReplyTo = &req.ReplyTo
	}
	if len(req.AtUsers) > 0 {
		grpcReq.AtUsers = req.AtUsers
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.clientManager.Message().SendMessage(ctx, grpcReq)
	if err != nil {
		logger.Error("Failed to send message via gRPC",
			zap.String("userID", c.UserID),
			zap.Error(err))
		return
	}

	var ts int64
	if resp.Timestamp != nil {
		ts = resp.Timestamp.Seconds
	}

	result := &sendMessageResult{
		MessageID: resp.MessageId,
		Sequence:  resp.Sequence,
		Timestamp: ts,
		LocalID:   req.LocalID,
	}

	resultData, _ := json.Marshal(result)
	wsResp := &websocket.Message{
		Type:    "message.sent",
		Payload: json.RawMessage(resultData),
	}
	h.wsManager.SendMessageToUser(c.UserID, wsResp)
}
