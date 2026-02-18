package model

import "time"

// GroupJoinRequest 入群申请模型
type GroupJoinRequest struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	GroupID   string    `gorm:"column:group_id;not null" json:"groupId"`
	UserID    string    `gorm:"column:user_id;not null" json:"userId"`
	InviterID string    `gorm:"column:inviter_id" json:"inviterId"` // NULL表示主动申请
	Message   string    `gorm:"column:message;size:200" json:"message"`
	Status    string    `gorm:"column:status;default:pending;size:20" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName 表名
func (GroupJoinRequest) TableName() string {
	return "group_join_requests"
}

// JoinRequestStatus 入群申请状态
const (
	JoinRequestStatusPending  = "pending"  // 待处理
	JoinRequestStatusAccepted = "accepted" // 已接受
	JoinRequestStatusRejected = "rejected" // 已拒绝
)

// IsPending 是否待处理
func (r *GroupJoinRequest) IsPending() bool {
	return r.Status == JoinRequestStatusPending
}

// IsProcessed 是否已处理
func (r *GroupJoinRequest) IsProcessed() bool {
	return r.Status == JoinRequestStatusAccepted || r.Status == JoinRequestStatusRejected
}

// IsInvitation 是否是邀请（而非主动申请）
func (r *GroupJoinRequest) IsInvitation() bool {
	return r.InviterID != ""
}
