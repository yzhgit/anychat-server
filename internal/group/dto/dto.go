package dto

import "time"

// ========== Request DTOs ==========

// CreateGroupRequest 创建群组请求
type CreateGroupRequest struct {
	Name       string   `json:"name" binding:"required,min=1,max=100" example:"技术交流群"`
	Avatar     string   `json:"avatar" binding:"omitempty,url" example:"https://example.com/avatar.jpg"`
	MemberIDs  []string `json:"memberIds" binding:"required,min=1" example:"user-123,user-456"`
	JoinVerify bool     `json:"joinVerify" example:"true"`
}

// UpdateGroupRequest 更新群信息请求
type UpdateGroupRequest struct {
	Name         *string `json:"name,omitempty" binding:"omitempty,min=1,max=100" example:"新群名称"`
	Avatar       *string `json:"avatar,omitempty" binding:"omitempty,url" example:"https://example.com/new-avatar.jpg"`
	Announcement *string `json:"announcement,omitempty" binding:"omitempty,max=1000" example:"群公告内容"`
	JoinVerify   *bool   `json:"joinVerify,omitempty" example:"false"`
}

// InviteMembersRequest 邀请成员请求
type InviteMembersRequest struct {
	UserIDs []string `json:"userIds" binding:"required,min=1" example:"user-123,user-456"`
}

// UpdateMemberRoleRequest 更新成员角色请求
type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member" example:"admin"`
}

// UpdateMemberNicknameRequest 更新群昵称请求
type UpdateMemberNicknameRequest struct {
	Nickname string `json:"nickname" binding:"required,max=50" example:"群内昵称"`
}

// TransferOwnershipRequest 转让群主请求
type TransferOwnershipRequest struct {
	NewOwnerID string `json:"newOwnerId" binding:"required" example:"user-456"`
}

// JoinGroupRequest 加入群组请求
type JoinGroupRequest struct {
	Message string `json:"message" binding:"max=200" example:"你好，我想加入群组"`
}

// HandleJoinRequestRequest 处理入群申请请求
type HandleJoinRequestRequest struct {
	Accept bool `json:"accept" example:"true"`
}

// UpdateGroupSettingsRequest 更新群组设置请求
type UpdateGroupSettingsRequest struct {
	AllowMemberInvite  *bool `json:"allowMemberInvite,omitempty" example:"true"`
	AllowViewHistory   *bool `json:"allowViewHistory,omitempty" example:"true"`
	AllowAddFriend     *bool `json:"allowAddFriend,omitempty" example:"true"`
	ShowMemberNickname *bool `json:"showMemberNickname,omitempty" example:"true"`
}

// ========== Response DTOs ==========

// GroupResponse 群组信息响应
type GroupResponse struct {
	GroupID      string    `json:"groupId" example:"group-123"`
	Name         string    `json:"name" example:"技术交流群"`
	Avatar       string    `json:"avatar" example:"https://example.com/avatar.jpg"`
	Announcement string    `json:"announcement" example:"欢迎加入"`
	OwnerID      string    `json:"ownerId" example:"user-123"`
	MemberCount  int32     `json:"memberCount" example:"10"`
	MaxMembers   int32     `json:"maxMembers" example:"500"`
	JoinVerify   bool      `json:"joinVerify" example:"true"`
	IsMuted      bool      `json:"isMuted" example:"false"`
	MyRole       string    `json:"myRole,omitempty" example:"member"`
	CreatedAt    time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt    time.Time `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// GroupListResponse 群组列表响应
type GroupListResponse struct {
	Groups     []*GroupResponse `json:"groups"`
	Total      int64            `json:"total" example:"5"`
	UpdateTime int64            `json:"updateTime,omitempty" example:"1609459200"`
}

// GroupMemberResponse 群成员响应
type GroupMemberResponse struct {
	UserID        string    `json:"userId" example:"user-123"`
	GroupNickname *string   `json:"groupNickname,omitempty" example:"群内昵称"`
	Role          string    `json:"role" example:"member"`
	IsMuted       bool      `json:"isMuted" example:"false"`
	JoinedAt      time.Time `json:"joinedAt" example:"2024-01-01T00:00:00Z"`
	UserInfo      *UserInfo `json:"userInfo,omitempty"`
}

// GroupMemberListResponse 群成员列表响应
type GroupMemberListResponse struct {
	Members []*GroupMemberResponse `json:"members"`
	Total   int64                  `json:"total" example:"10"`
	Page    int                    `json:"page" example:"1"`
}

// JoinGroupResponse 加入群组响应
type JoinGroupResponse struct {
	NeedVerify bool   `json:"needVerify" example:"true"`
	RequestID  *int64 `json:"requestId,omitempty" example:"123"`
	Message    string `json:"message" example:"申请已提交，等待审核"`
}

// JoinRequestResponse 入群申请响应
type JoinRequestResponse struct {
	ID        int64     `json:"id" example:"123"`
	GroupID   string    `json:"groupId" example:"group-123"`
	UserID    string    `json:"userId" example:"user-123"`
	InviterID *string   `json:"inviterId,omitempty" example:"user-456"`
	Message   string    `json:"message" example:"你好，我想加入"`
	Status    string    `json:"status" example:"pending"`
	CreatedAt time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UserInfo  *UserInfo `json:"userInfo,omitempty"`
}

// JoinRequestListResponse 入群申请列表响应
type JoinRequestListResponse struct {
	Requests []*JoinRequestResponse `json:"requests"`
	Total    int64                  `json:"total" example:"3"`
}

// GroupSettingsResponse 群组设置响应
type GroupSettingsResponse struct {
	GroupID            string `json:"groupId" example:"group-123"`
	AllowMemberInvite  bool   `json:"allowMemberInvite" example:"true"`
	AllowViewHistory   bool   `json:"allowViewHistory" example:"true"`
	AllowAddFriend     bool   `json:"allowAddFriend" example:"true"`
	ShowMemberNickname bool   `json:"showMemberNickname" example:"true"`
}

// ========== Shared Types ==========

// UserInfo 用户基本信息（从user-service获取）
type UserInfo struct {
	UserID   string  `json:"userId" example:"user-123"`
	Nickname string  `json:"nickname" example:"张三"`
	Avatar   string  `json:"avatar" example:"https://example.com/avatar.jpg"`
	Gender   *int32  `json:"gender,omitempty" example:"1"`
	Bio      *string `json:"bio,omitempty" example:"个性签名"`
}
