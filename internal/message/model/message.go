package model

import (
	"time"
)

// Message 消息模型
type Message struct {
	ID               int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MessageID        string    `gorm:"column:message_id;not null;uniqueIndex" json:"messageId"`
	ConversationID   string    `gorm:"column:conversation_id;not null;index:idx_conversation_sequence" json:"conversationId"`
	ConversationType string    `gorm:"column:conversation_type;not null" json:"conversationType"` // single/group
	SenderID         string    `gorm:"column:sender_id;not null;index:idx_sender_time" json:"senderId"`
	ContentType      string    `gorm:"column:content_type;not null" json:"contentType"` // text/image/video/audio/file/location/card
	Content          string    `gorm:"column:content;type:jsonb;not null" json:"content"`
	Sequence         int64     `gorm:"column:sequence;not null;uniqueIndex:uk_conversation_sequence" json:"sequence"`
	ReplyTo          *string   `gorm:"column:reply_to" json:"replyTo,omitempty"`
	AtUsers          []string  `gorm:"column:at_users;type:text[]" json:"atUsers,omitempty"`
	Status           int16     `gorm:"column:status;default:0" json:"status"` // 0-正常 1-撤回 2-删除
	CreatedAt        time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP;index:idx_created_at" json:"createdAt"`
	UpdatedAt        time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName 表名
func (Message) TableName() string {
	return "messages"
}

// MessageStatus 消息状态
const (
	MessageStatusNormal  = 0 // 正常
	MessageStatusRecall  = 1 // 撤回
	MessageStatusDeleted = 2 // 删除
)

// ConversationType 会话类型
const (
	ConversationTypeSingle = "single" // 单聊
	ConversationTypeGroup  = "group"  // 群聊
)

// ContentType 内容类型
const (
	ContentTypeText     = "text"     // 文本
	ContentTypeImage    = "image"    // 图片
	ContentTypeVideo    = "video"    // 视频
	ContentTypeAudio    = "audio"    // 语音
	ContentTypeFile     = "file"     // 文件
	ContentTypeLocation = "location" // 位置
	ContentTypeCard     = "card"     // 名片
)

// IsNormal 是否正常消息
func (m *Message) IsNormal() bool {
	return m.Status == MessageStatusNormal
}

// IsRecalled 是否已撤回
func (m *Message) IsRecalled() bool {
	return m.Status == MessageStatusRecall
}

// IsDeleted 是否已删除
func (m *Message) IsDeleted() bool {
	return m.Status == MessageStatusDeleted
}
