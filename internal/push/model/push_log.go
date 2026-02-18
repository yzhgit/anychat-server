package model

import "time"

// PushLog 推送日志模型
type PushLog struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	UserID       string    `gorm:"column:user_id;not null;index"`
	PushType     string    `gorm:"column:push_type;not null"`
	Title        string    `gorm:"column:title"`
	Content      string    `gorm:"column:content"`
	TargetCount  int       `gorm:"column:target_count;not null;default:0"`
	SuccessCount int       `gorm:"column:success_count;not null;default:0"`
	FailureCount int       `gorm:"column:failure_count;not null;default:0"`
	JPushMsgID   string    `gorm:"column:jpush_msg_id"`
	Status       string    `gorm:"column:status;not null;default:pending"` // pending/sent/failed
	ErrorMsg     string    `gorm:"column:error_msg"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (PushLog) TableName() string {
	return "push_logs"
}
