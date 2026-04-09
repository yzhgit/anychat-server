package notification

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Priority notification priority
type Priority string

const (
	PriorityHigh   Priority = "high"   // High priority
	PriorityNormal Priority = "normal" // Normal priority
	PriorityLow    Priority = "low"    // Low priority
)

// Notification common notification structure
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

// Publisher notification publisher interface
type Publisher interface {
	Publish(notification *Notification) error
	PublishToUser(userID string, notification *Notification) error
	PublishToUsers(userIDs []string, notification *Notification) error
	PublishToGroup(groupID string, notification *Notification) error
	PublishBroadcast(notification *Notification) error
}

// natsPublisher NATS notification publisher implementation
type natsPublisher struct {
	nc *nats.Conn
}

// NewPublisher creates a new NATS notification publisher
func NewPublisher(nc *nats.Conn) Publisher {
	return &natsPublisher{nc: nc}
}

// Publish publishes a notification (generic method, requires toUserID in notification)
func (p *natsPublisher) Publish(notification *Notification) error {
	if notification.ToUserID == "" {
		return fmt.Errorf("toUserID is required for publishing notification")
	}
	return p.PublishToUser(notification.ToUserID, notification)
}

// PublishToUser publishes a notification to a specific user
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

// PublishToUsers batch publishes notifications to multiple users
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

// PublishToGroup publishes a group notification (will be sent to all users subscribed to the group)
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

// PublishBroadcast broadcasts a notification to all users
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

// BuildUserNotificationSubject builds user notification subject
// Format: notification.{service}.{event_type}.{user_id}
// Example: notification.friend.request.user-123
func BuildUserNotificationSubject(notificationType, userID string) string {
	return fmt.Sprintf("notification.%s.%s", notificationType, userID)
}

// BuildGroupNotificationSubject builds group notification subject
// Format: notification.{service}.{event_type}.{group_id}
// Example: notification.group.member_joined.group-456
func BuildGroupNotificationSubject(notificationType, groupID string) string {
	return fmt.Sprintf("notification.%s.%s", notificationType, groupID)
}

// BuildBroadcastSubject builds broadcast notification subject
// Format: notification.{service}.{event_type}.broadcast
// Example: notification.admin.announcement.broadcast
func BuildBroadcastSubject(notificationType string) string {
	return fmt.Sprintf("notification.%s.broadcast", notificationType)
}
