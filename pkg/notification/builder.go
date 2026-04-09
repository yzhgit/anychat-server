package notification

import (
	"github.com/google/uuid"
)

// NewNotification creates a new notification
func NewNotification(notifType string, fromUserID string, priority Priority) *Notification {
	return &Notification{
		NotificationID: uuid.New().String(),
		Type:           notifType,
		FromUserID:     fromUserID,
		Priority:       priority,
		Payload:        make(map[string]interface{}),
		Metadata:       make(map[string]interface{}),
	}
}

// WithPayload sets payload
func (n *Notification) WithPayload(payload map[string]interface{}) *Notification {
	n.Payload = payload
	return n
}

// WithMetadata sets metadata
func (n *Notification) WithMetadata(metadata map[string]interface{}) *Notification {
	n.Metadata = metadata
	return n
}

// AddPayloadField adds payload field
func (n *Notification) AddPayloadField(key string, value interface{}) *Notification {
	if n.Payload == nil {
		n.Payload = make(map[string]interface{})
	}
	n.Payload[key] = value
	return n
}

// AddMetadataField adds metadata field
func (n *Notification) AddMetadataField(key string, value interface{}) *Notification {
	if n.Metadata == nil {
		n.Metadata = make(map[string]interface{})
	}
	n.Metadata[key] = value
	return n
}
