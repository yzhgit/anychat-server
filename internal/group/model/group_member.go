package model

import "time"

// GroupMember 群成员模型
type GroupMember struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	GroupID       string    `gorm:"column:group_id;not null;uniqueIndex:uk_group_user" json:"groupId"`
	UserID        string    `gorm:"column:user_id;not null;uniqueIndex:uk_group_user" json:"userId"`
	GroupNickname string    `gorm:"column:group_nickname;size:50" json:"groupNickname"`
	Role          string    `gorm:"column:role;default:member;size:20" json:"role"`
	IsMuted       bool      `gorm:"column:is_muted;default:false" json:"isMuted"`
	JoinedAt      time.Time `gorm:"column:joined_at;not null;default:CURRENT_TIMESTAMP" json:"joinedAt"`
	UpdatedAt     time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName 表名
func (GroupMember) TableName() string {
	return "group_members"
}

// GroupRole 群成员角色
const (
	GroupRoleOwner  = "owner"  // 群主
	GroupRoleAdmin  = "admin"  // 管理员
	GroupRoleMember = "member" // 普通成员
)

// IsOwner 是否是群主
func (gm *GroupMember) IsOwner() bool {
	return gm.Role == GroupRoleOwner
}

// IsAdmin 是否是管理员（包括群主）
func (gm *GroupMember) IsAdmin() bool {
	return gm.Role == GroupRoleOwner || gm.Role == GroupRoleAdmin
}

// CanManageGroup 是否可以管理群组（群主和管理员）
func (gm *GroupMember) CanManageGroup() bool {
	return gm.IsAdmin()
}

// CanRemoveMember 是否可以移除指定角色的成员
func (gm *GroupMember) CanRemoveMember(targetRole string) bool {
	if gm.IsOwner() {
		// 群主可以移除任何人（除了自己）
		return true
	}
	if gm.Role == GroupRoleAdmin {
		// 管理员只能移除普通成员
		return targetRole == GroupRoleMember
	}
	return false
}
