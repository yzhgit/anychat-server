package model

import "time"

// GroupPinnedMessage 群置顶消息
type GroupPinnedMessage struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	GroupID     string    `gorm:"column:group_id;not null;index:idx_group_pinned_messages_group_id" json:"groupId"`
	MessageID   string    `gorm:"column:message_id;not null" json:"messageId"`
	MessageSeq  *int64    `gorm:"column:message_seq" json:"messageSeq,omitempty"`
	PinnedBy    string    `gorm:"column:pinned_by;not null" json:"pinnedBy"`
	Content     string    `gorm:"column:content;type:text" json:"content"`
	ContentType string    `gorm:"column:content_type;type:varchar(32);not null;default:text" json:"contentType"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (GroupPinnedMessage) TableName() string {
	return "group_pinned_messages"
}
