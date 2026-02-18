package model

import (
	"time"
)

// Session 会话模型
type Session struct {
	SessionID          string     `gorm:"column:session_id;primaryKey"`
	SessionType        string     `gorm:"column:session_type;not null"`        // single/group/system
	UserID             string     `gorm:"column:user_id;not null"`
	TargetID           string     `gorm:"column:target_id;not null"`
	LastMessageID      string     `gorm:"column:last_message_id"`
	LastMessageContent string     `gorm:"column:last_message_content"`
	LastMessageTime    *time.Time `gorm:"column:last_message_time"`
	UnreadCount        int32      `gorm:"column:unread_count;default:0"`
	IsPinned           bool       `gorm:"column:is_pinned;default:false"`
	IsMuted            bool       `gorm:"column:is_muted;default:false"`
	PinTime            *time.Time `gorm:"column:pin_time"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName 指定表名
func (Session) TableName() string {
	return "sessions"
}

// SessionType 会话类型常量
const (
	SessionTypeSingle = "single"
	SessionTypeGroup  = "group"
	SessionTypeSystem = "system"
)
