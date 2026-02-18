package model

import "time"

// MessageReference 消息引用关系
type MessageReference struct {
	ID               int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MessageID        string    `gorm:"column:message_id;not null;uniqueIndex:uk_message_reply;index:idx_message" json:"messageId"`
	ReplyToMessageID string    `gorm:"column:reply_to_message_id;not null;uniqueIndex:uk_message_reply;index:idx_reply_to" json:"replyToMessageId"`
	CreatedAt        time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
}

// TableName 表名
func (MessageReference) TableName() string {
	return "message_references"
}
