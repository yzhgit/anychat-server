package model

import "time"

// ConversationSequence 会话序列号
type ConversationSequence struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ConversationID string    `gorm:"column:conversation_id;not null;uniqueIndex" json:"conversationId"`
	CurrentSeq     int64     `gorm:"column:current_seq;not null;default:0" json:"currentSeq"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName 表名
func (ConversationSequence) TableName() string {
	return "conversation_sequences"
}
