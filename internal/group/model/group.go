package model

import "time"

// Group 群组模型
type Group struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	GroupID      string    `gorm:"column:group_id;not null;uniqueIndex" json:"groupId"`
	Name         string    `gorm:"column:name;not null;size:100" json:"name"`
	Avatar       string    `gorm:"column:avatar;size:255" json:"avatar"`
	Announcement string    `gorm:"column:announcement;type:text" json:"announcement"`
	OwnerID      string    `gorm:"column:owner_id;not null" json:"ownerId"`
	MemberCount  int32     `gorm:"column:member_count;default:0" json:"memberCount"`
	MaxMembers   int32     `gorm:"column:max_members;default:500" json:"maxMembers"`
	JoinVerify   bool      `gorm:"column:join_verify;default:true" json:"joinVerify"`
	IsMuted      bool      `gorm:"column:is_muted;default:false" json:"isMuted"`
	Status       int16     `gorm:"column:status;default:1" json:"status"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName 表名
func (Group) TableName() string {
	return "groups"
}

// GroupStatus 群组状态
const (
	GroupStatusDissolved = 0 // 已解散
	GroupStatusNormal    = 1 // 正常
)

// IsActive 是否有效
func (g *Group) IsActive() bool {
	return g.Status == GroupStatusNormal
}

// IsFull 是否已满
func (g *Group) IsFull() bool {
	return g.MemberCount >= g.MaxMembers
}
