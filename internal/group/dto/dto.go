package dto

import "time"

// ========== Request DTOs ==========

// CreateGroupRequest 创建群组请求
type CreateGroupRequest struct {
	Name      string   `json:"name" binding:"required,min=1,max=100" example:"技术交流群"`
	Avatar    string   `json:"avatar" binding:"omitempty,url" example:"https://example.com/avatar.jpg"`
	MemberIDs []string `json:"memberIds" binding:"required,min=1" example:"user-123,user-456"`
}

// UpdateGroupRequest 更新群信息请求
type UpdateGroupRequest struct {
	Name         *string `json:"name,omitempty" binding:"omitempty,min=1,max=100" example:"新群名称"`
	Avatar       *string `json:"avatar,omitempty" binding:"omitempty,url" example:"https://example.com/new-avatar.jpg"`
	Announcement *string `json:"announcement,omitempty" binding:"omitempty,max=1000" example:"群公告内容"`
	Description  *string `json:"description,omitempty" binding:"omitempty,max=1000" example:"群简介内容"`
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

// UpdateMemberRemarkRequest 更新群备注请求（仅对自己可见）
type UpdateMemberRemarkRequest struct {
	Remark string `json:"remark" binding:"max=20" example:"产品讨论群"` // 空字符串表示清空备注
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
	JoinVerify        *bool `json:"joinVerify,omitempty" example:"true"`
	AllowMemberInvite *bool `json:"allowMemberInvite,omitempty" example:"true"`
	AllowViewHistory  *bool `json:"allowViewHistory,omitempty" example:"true"`
	AllowAddFriend    *bool `json:"allowAddFriend,omitempty" example:"true"`
	AllowMemberModify *bool `json:"allowMemberModify,omitempty" example:"false"`
}

// PinGroupMessageRequest 置顶群消息请求
type PinGroupMessageRequest struct {
	MessageID string `json:"messageId" binding:"required" example:"msg-123"`
	Content   string `json:"content,omitempty" example:"消息摘要"`
}

// SetGroupMuteRequest 设置全体禁言请求
type SetGroupMuteRequest struct {
	Enabled *bool `json:"enabled" binding:"required" example:"true"`
}

// MuteMemberRequest 禁言成员请求
type MuteMemberRequest struct {
	Type            string `json:"type" binding:"required,oneof=permanent temporary" example:"temporary"`
	DurationMinutes int32  `json:"durationMinutes,omitempty" example:"60"`
}

// ========== Response DTOs ==========

// GroupResponse 群组信息响应
type GroupResponse struct {
	GroupID      string    `json:"groupId" example:"group-123"`
	Name         string    `json:"name" example:"技术交流群"`
	DisplayName  string    `json:"displayName" example:"产品讨论群"` // 群备注优先，无备注则同 Name
	Avatar       string    `json:"avatar" example:"https://example.com/avatar.jpg"`
	Announcement string    `json:"announcement" example:"欢迎加入"`
	Description  string    `json:"description" example:"群简介"`
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
	UserID        string     `json:"userId" example:"user-123"`
	GroupNickname *string    `json:"groupNickname,omitempty" example:"群内昵称"`
	Role          string     `json:"role" example:"member"`
	IsMuted       bool       `json:"isMuted" example:"false"`
	MutedUntil    *time.Time `json:"mutedUntil,omitempty" example:"2024-01-01T01:00:00Z"`
	JoinedAt      time.Time  `json:"joinedAt" example:"2024-01-01T00:00:00Z"`
	UserInfo      *UserInfo  `json:"userInfo,omitempty"`
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
	GroupID           string `json:"groupId" example:"group-123"`
	JoinVerify        bool   `json:"joinVerify" example:"true"`
	AllowMemberInvite bool   `json:"allowMemberInvite" example:"true"`
	AllowViewHistory  bool   `json:"allowViewHistory" example:"true"`
	AllowAddFriend    bool   `json:"allowAddFriend" example:"true"`
	AllowMemberModify bool   `json:"allowMemberModify" example:"false"`
}

// PinnedMessageResponse 群置顶消息
type PinnedMessageResponse struct {
	MessageID string `json:"messageId" example:"msg-123"`
	Content   string `json:"content" example:"消息摘要"`
	PinnedBy  string `json:"pinnedBy" example:"user-123"`
	PinnedAt  int64  `json:"pinnedAt" example:"1710000000"`
}

// PinnedMessageListResponse 群置顶消息列表
type PinnedMessageListResponse struct {
	Messages []*PinnedMessageResponse `json:"messages"`
}

// GroupQRCodeResponse 群二维码响应
type GroupQRCodeResponse struct {
	Token    string `json:"token" example:"550e8400-e29b-41d4-a716-446655440000"`
	DeepLink string `json:"deepLink" example:"anychat://join/group?token=550e8400-e29b-41d4-a716-446655440000"`
	ExpireAt int64  `json:"expireAt" example:"1754006400"` // Unix 时间戳（秒）
}

// GroupQRCodePreviewResponse 二维码群信息预览响应
type GroupQRCodePreviewResponse struct {
	GroupID     string `json:"groupId" example:"group-123"`
	Name        string `json:"name" example:"产品团队"`
	Avatar      string `json:"avatar" example:"https://example.com/group.png"`
	MemberCount int32  `json:"memberCount" example:"42"`
	NeedVerify  bool   `json:"needVerify" example:"true"`
}

// JoinGroupByQRCodeResponse 扫码加入群响应
type JoinGroupByQRCodeResponse struct {
	Joined     bool   `json:"joined" example:"true"` // true=直接加入，false=已提交申请
	GroupID    string `json:"groupId" example:"group-123"`
	NeedVerify bool   `json:"needVerify" example:"false"` // 是否需要审批
	RequestID  *int64 `json:"requestId,omitempty" example:"123"`
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
