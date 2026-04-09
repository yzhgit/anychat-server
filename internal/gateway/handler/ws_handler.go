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
	// Allow all origins (should verify Origin in production)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WSHandler WebSocket handler
type WSHandler struct {
	clientManager *client.Manager
	jwtManager    *jwt.Manager
	wsManager     *websocket.Manager
	subscriber    *gwnotification.Subscriber
}

// NewWSHandler creates WebSocket handler
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

// HandleWebSocket handle WebSocket connection
// @Summary      establish WebSocket long connection
// @Description  Client maintains long connection via WebSocket to receive real-time notifications and message pushes. Since WebSocket protocol doesn't support custom headers, JWT token is passed via URL query parameter.
// @Tags         realtime
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

	// ReadPump blocks until connection disconnects
	wsClient.ReadPump(h.handleClientMessage)

	// After connection disconnects, only unsubscribe from NATS when user is truly offline
	// IsOnline returns false when user has no new active connections (not replaced)
	if !h.wsManager.IsOnline(userID) {
		h.subscriber.UnsubscribeUser(userID)
	}

	logger.Info("WebSocket client disconnected",
		zap.String("userID", userID),
		zap.Int("onlineCount", h.wsManager.OnlineCount()))
}

// handleClientMessage handle messages from WebSocket client
func (h *WSHandler) handleClientMessage(c *websocket.Client, msg *websocket.Message) {
	switch msg.Type {
	case "ping":
		pong := &websocket.Message{Type: "pong"}
		h.wsManager.SendMessageToUser(c.UserID, pong)

	case "message.send":
		h.handleSendMessage(c, msg.Payload)

	case "message.typing":
		h.handleSendTyping(c, msg.Payload)

	default:
		logger.Debug("Unknown WebSocket message type",
			zap.String("type", msg.Type),
			zap.String("userID", c.UserID))
	}
}

// sendMessagePayload payload structure for client sending messages
type sendMessagePayload struct {
	ConversationID string   `json:"conversationId"`
	ContentType    string   `json:"contentType"`
	Content        string   `json:"content"`
	ReplyTo        string   `json:"replyTo,omitempty"`
	AtUsers        []string `json:"atUsers,omitempty"`
	LocalID        string   `json:"localId,omitempty"`
}

type sendTypingPayload struct {
	ConversationID string `json:"conversationId"`
	Typing         *bool  `json:"typing"`
	TTLSeconds     *int32 `json:"ttlSeconds,omitempty"`
	ClientTs       *int64 `json:"clientTs,omitempty"`
}

// sendMessageResult response structure for sending messages
type sendMessageResult struct {
	MessageID string `json:"messageId"`
	Sequence  int64  `json:"sequence"`
	Timestamp int64  `json:"timestamp"`
	LocalID   string `json:"localId,omitempty"`
}

type sendMessageError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	LocalID string `json:"localId,omitempty"`
}

type sendTypingError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// handleSendMessage parse message.send payload and forward via gRPC
func (h *WSHandler) handleSendMessage(c *websocket.Client, payload json.RawMessage) {
	var req sendMessagePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		logger.Warn("Invalid message.send payload",
			zap.String("userID", c.UserID),
			zap.Error(err))
		return
	}

	grpcReq := &messagepb.SendMessageRequest{
		SenderId:       c.UserID,
		ConversationId: req.ConversationID,
		ContentType:    req.ContentType,
		Content:        req.Content,
		LocalId:        req.LocalID,
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

		wsErr := &sendMessageError{
			Code:    "send_failed",
			Message: err.Error(),
			LocalID: req.LocalID,
		}
		errData, _ := json.Marshal(wsErr)
		h.wsManager.SendMessageToUser(c.UserID, &websocket.Message{
			Type:    "message.error",
			Payload: json.RawMessage(errData),
		})
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

func (h *WSHandler) handleSendTyping(c *websocket.Client, payload json.RawMessage) {
	var req sendTypingPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		logger.Warn("Invalid message.typing payload",
			zap.String("userID", c.UserID),
			zap.Error(err))
		return
	}
	if req.ConversationID == "" || req.Typing == nil {
		logger.Warn("Invalid message.typing payload fields",
			zap.String("userID", c.UserID),
			zap.String("conversationID", req.ConversationID))
		return
	}

	grpcReq := &messagepb.SendTypingRequest{
		ConversationId: req.ConversationID,
		FromUserId:     c.UserID,
		Typing:         *req.Typing,
	}
	if req.TTLSeconds != nil {
		grpcReq.TtlSeconds = req.TTLSeconds
	}
	if c.DeviceID != "" {
		grpcReq.DeviceId = &c.DeviceID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := h.clientManager.Message().SendTyping(ctx, grpcReq); err != nil {
		logger.Error("Failed to send typing via gRPC",
			zap.String("userID", c.UserID),
			zap.Error(err))

		wsErr := &sendTypingError{
			Code:    "typing_failed",
			Message: err.Error(),
		}
		errData, _ := json.Marshal(wsErr)
		h.wsManager.SendMessageToUser(c.UserID, &websocket.Message{
			Type:    "message.error",
			Payload: json.RawMessage(errData),
		})
	}
}
