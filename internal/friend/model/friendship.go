package model

import "time"

// Friendship 好友关系模型
type Friendship struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    string    `gorm:"column:user_id;not null;uniqueIndex:uk_user_friend" json:"userId"`
	FriendID  string    `gorm:"column:friend_id;not null;uniqueIndex:uk_user_friend" json:"friendId"`
	Remark    string    `gorm:"column:remark;size:50" json:"remark"`
	Status    int16     `gorm:"column:status;default:1" json:"status"` // 0-已删除 1-正常
	CreatedAt time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName 表名
func (Friendship) TableName() string {
	return "friendships"
}

// FriendshipStatus 好友关系状态
const (
	FriendshipStatusDeleted = 0 // 已删除
	FriendshipStatusNormal  = 1 // 正常
)

// IsActive 是否有效
func (f *Friendship) IsActive() bool {
	return f.Status == FriendshipStatusNormal
}
