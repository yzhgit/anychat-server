package model

import "time"

// FriendRequest is the friend request model
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

// TableName is the table name
func (FriendRequest) TableName() string {
	return "friend_requests"
}

// FriendRequestStatus is the friend request status constant
const (
	FriendRequestStatusPending  = "pending"  // pending
	FriendRequestStatusAccepted = "accepted" // accepted
	FriendRequestStatusRejected = "rejected" // rejected
	FriendRequestStatusExpired  = "expired"  // expired
)

// FriendRequestSource is the friend request source constant
const (
	FriendRequestSourceSearch   = "search"   // search
	FriendRequestSourceQRCode   = "qrcode"   // qrcode
	FriendRequestSourceGroup    = "group"    // group
	FriendRequestSourceContacts = "contacts" // contacts
)

// IsPending returns true if the request is pending
func (fr *FriendRequest) IsPending() bool {
	return fr.Status == FriendRequestStatusPending
}

// IsAccepted returns true if the request is accepted
func (fr *FriendRequest) IsAccepted() bool {
	return fr.Status == FriendRequestStatusAccepted
}

// IsRejected returns true if the request is rejected
func (fr *FriendRequest) IsRejected() bool {
	return fr.Status == FriendRequestStatusRejected
}
