package notification

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/anychat/server/internal/gateway/websocket"
	"github.com/anychat/server/pkg/logger"
	pkgnotification "github.com/anychat/server/pkg/notification"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Subscriber NATS订阅管理器，负责订阅用户通知并推送给WebSocket客户端
type Subscriber struct {
	nc      *nats.Conn
	manager *websocket.Manager
	subs    map[string]*nats.Subscription // userID -> subscription
	mu      sync.RWMutex
}

// NewSubscriber 创建NATS订阅管理器
func NewSubscriber(nc *nats.Conn, manager *websocket.Manager) *Subscriber {
	return &Subscriber{
		nc:      nc,
		manager: manager,
		subs:    make(map[string]*nats.Subscription),
	}
}

// SubscribeUser 为用户订阅NATS通知（幂等，已订阅则跳过）
// 订阅主题: notification.*.*.{userID}，匹配该用户的所有服务通知
func (s *Subscriber) SubscribeUser(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.subs[userID]; exists {
		return nil
	}

	subject := fmt.Sprintf("notification.*.*.%s", userID)
	sub, err := s.nc.Subscribe(subject, func(msg *nats.Msg) {
		s.handleNotification(userID, msg.Data)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe for user %s: %w", userID, err)
	}

	s.subs[userID] = sub
	logger.Info("Subscribed to user notifications",
		zap.String("userID", userID),
		zap.String("subject", subject))
	return nil
}

// UnsubscribeUser 取消用户的NATS通知订阅
func (s *Subscriber) UnsubscribeUser(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sub, exists := s.subs[userID]; exists {
		if err := sub.Unsubscribe(); err != nil {
			logger.Warn("Failed to unsubscribe",
				zap.String("userID", userID),
				zap.Error(err))
		}
		delete(s.subs, userID)
		logger.Info("Unsubscribed from user notifications", zap.String("userID", userID))
	}
}

// handleNotification 处理收到的NATS通知，序列化后推送给WebSocket客户端
func (s *Subscriber) handleNotification(userID string, data []byte) {
	var notif pkgnotification.Notification
	if err := json.Unmarshal(data, &notif); err != nil {
		logger.Error("Failed to parse notification", zap.Error(err))
		return
	}

	payload, err := json.Marshal(&notif)
	if err != nil {
		logger.Error("Failed to marshal notification for WebSocket", zap.Error(err))
		return
	}

	wsMsg := &websocket.Message{
		Type:    "notification",
		Payload: json.RawMessage(payload),
	}

	if !s.manager.SendMessageToUser(userID, wsMsg) {
		logger.Debug("User not connected, notification dropped",
			zap.String("userID", userID),
			zap.String("type", notif.Type))
	}
}
