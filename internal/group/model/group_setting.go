package model

import "time"

// GroupSetting 群组设置模型
type GroupSetting struct {
	GroupID            string    `gorm:"column:group_id;primaryKey" json:"groupId"`
	AllowMemberInvite  bool      `gorm:"column:allow_member_invite;default:true" json:"allowMemberInvite"`
	AllowViewHistory   bool      `gorm:"column:allow_view_history;default:true" json:"allowViewHistory"`
	AllowAddFriend     bool      `gorm:"column:allow_add_friend;default:true" json:"allowAddFriend"`
	ShowMemberNickname bool      `gorm:"column:show_member_nickname;default:true" json:"showMemberNickname"`
	CreatedAt          time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt          time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName 表名
func (GroupSetting) TableName() string {
	return "group_settings"
}

// DefaultGroupSetting 创建默认群组设置
func DefaultGroupSetting(groupID string) *GroupSetting {
	return &GroupSetting{
		GroupID:            groupID,
		AllowMemberInvite:  true,
		AllowViewHistory:   true,
		AllowAddFriend:     true,
		ShowMemberNickname: true,
	}
}
