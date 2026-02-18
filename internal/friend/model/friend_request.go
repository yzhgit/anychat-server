package model

import "time"

// FriendRequest 好友申请模型
type FriendRequest struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	FromUserID string    `gorm:"column:from_user_id;not null;index" json:"fromUserId"`
	ToUserID   string    `gorm:"column:to_user_id;not null;index:idx_friend_requests_to_user" json:"toUserId"`
	Message    string    `gorm:"column:message;size:200" json:"message"`
	Source     string    `gorm:"column:source;size:20" json:"source"` // search/qrcode/group/contacts
	Status     string    `gorm:"column:status;size:20;default:'pending';index:idx_friend_requests_to_user" json:"status"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP;index" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName 表名
func (FriendRequest) TableName() string {
	return "friend_requests"
}

// FriendRequestStatus 好友申请状态
const (
	FriendRequestStatusPending  = "pending"  // 待处理
	FriendRequestStatusAccepted = "accepted" // 已接受
	FriendRequestStatusRejected = "rejected" // 已拒绝
	FriendRequestStatusExpired  = "expired"  // 已过期
)

// FriendRequestSource 好友申请来源
const (
	FriendRequestSourceSearch   = "search"   // 搜索
	FriendRequestSourceQRCode   = "qrcode"   // 二维码
	FriendRequestSourceGroup    = "group"    // 群组
	FriendRequestSourceContacts = "contacts" // 通讯录
)

// IsPending 是否待处理
func (fr *FriendRequest) IsPending() bool {
	return fr.Status == FriendRequestStatusPending
}

// IsAccepted 是否已接受
func (fr *FriendRequest) IsAccepted() bool {
	return fr.Status == FriendRequestStatusAccepted
}

// IsRejected 是否已拒绝
func (fr *FriendRequest) IsRejected() bool {
	return fr.Status == FriendRequestStatusRejected
}
