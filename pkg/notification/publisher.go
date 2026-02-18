package notification

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Priority 通知优先级
type Priority string

const (
	PriorityHigh   Priority = "high"   // 高优先级
	PriorityNormal Priority = "normal" // 普通优先级
	PriorityLow    Priority = "low"    // 低优先级
)

// Notification 通用通知结构
type Notification struct {
	NotificationID string                 `json:"notification_id"`
	Type           string                 `json:"type"`
	Timestamp      int64                  `json:"timestamp"`
	FromUserID     string                 `json:"from_user_id,omitempty"`
	ToUserID       string                 `json:"to_user_id,omitempty"`
	Priority       Priority               `json:"priority"`
	Payload        map[string]interface{} `json:"payload"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Publisher 通知发布器接口
type Publisher interface {
	Publish(notification *Notification) error
	PublishToUser(userID string, notification *Notification) error
	PublishToUsers(userIDs []string, notification *Notification) error
	PublishToGroup(groupID string, notification *Notification) error
	PublishBroadcast(notification *Notification) error
}

// natsPublisher NATS通知发布器实现
type natsPublisher struct {
	nc *nats.Conn
}

// NewPublisher 创建NATS通知发布器
func NewPublisher(nc *nats.Conn) Publisher {
	return &natsPublisher{nc: nc}
}

// Publish 发布通知（通用方法，需要在notification中指定toUserID）
func (p *natsPublisher) Publish(notification *Notification) error {
	if notification.ToUserID == "" {
		return fmt.Errorf("toUserID is required for publishing notification")
	}
	return p.PublishToUser(notification.ToUserID, notification)
}

// PublishToUser 发布通知给指定用户
func (p *natsPublisher) PublishToUser(userID string, notification *Notification) error {
	notification.ToUserID = userID
	if notification.Timestamp == 0 {
		notification.Timestamp = time.Now().Unix()
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	subject := BuildUserNotificationSubject(notification.Type, userID)
	return p.nc.Publish(subject, data)
}

// PublishToUsers 批量发布通知给多个用户
func (p *natsPublisher) PublishToUsers(userIDs []string, notification *Notification) error {
	if notification.Timestamp == 0 {
		notification.Timestamp = time.Now().Unix()
	}

	for _, userID := range userIDs {
		notification.ToUserID = userID
		if err := p.PublishToUser(userID, notification); err != nil {
			return err
		}
	}
	return nil
}

// PublishToGroup 发布群组通知（会发送给所有订阅该群组的用户）
func (p *natsPublisher) PublishToGroup(groupID string, notification *Notification) error {
	if notification.Timestamp == 0 {
		notification.Timestamp = time.Now().Unix()
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	subject := BuildGroupNotificationSubject(notification.Type, groupID)
	return p.nc.Publish(subject, data)
}

// PublishBroadcast 广播通知给所有用户
func (p *natsPublisher) PublishBroadcast(notification *Notification) error {
	if notification.Timestamp == 0 {
		notification.Timestamp = time.Now().Unix()
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	subject := BuildBroadcastSubject(notification.Type)
	return p.nc.Publish(subject, data)
}

// BuildUserNotificationSubject 构建用户通知主题
// 格式: notification.{service}.{event_type}.{user_id}
// 例如: notification.friend.request.user-123
func BuildUserNotificationSubject(notificationType, userID string) string {
	return fmt.Sprintf("notification.%s.%s", notificationType, userID)
}

// BuildGroupNotificationSubject 构建群组通知主题
// 格式: notification.{service}.{event_type}.{group_id}
// 例如: notification.group.member_joined.group-456
func BuildGroupNotificationSubject(notificationType, groupID string) string {
	return fmt.Sprintf("notification.%s.%s", notificationType, groupID)
}

// BuildBroadcastSubject 构建广播通知主题
// 格式: notification.{service}.{event_type}.broadcast
// 例如: notification.admin.announcement.broadcast
func BuildBroadcastSubject(notificationType string) string {
	return fmt.Sprintf("notification.%s.broadcast", notificationType)
}
