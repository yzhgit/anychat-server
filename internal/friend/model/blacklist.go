package model

import "time"

// Blacklist 黑名单模型
type Blacklist struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID        string    `gorm:"column:user_id;not null;uniqueIndex:uk_user_blocked;index" json:"userId"`
	BlockedUserID string    `gorm:"column:blocked_user_id;not null;uniqueIndex:uk_user_blocked;index" json:"blockedUserId"`
	CreatedAt     time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
}

// TableName 表名
func (Blacklist) TableName() string {
	return "blacklists"
}
