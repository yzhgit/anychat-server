package dto

import "time"

// SendFriendRequestRequest 发送好友申请请求
type SendFriendRequestRequest struct {
	UserID  string `json:"userId" binding:"required" example:"user-123"`
	Message string `json:"message" binding:"max=200" example:"你好，我想加你为好友"`
	Source  string `json:"source" binding:"required,oneof=search qrcode group contacts" example:"search"`
}

// HandleFriendRequestRequest 处理好友申请请求
type HandleFriendRequestRequest struct {
	Action string `json:"action" binding:"required,oneof=accept reject" example:"accept"`
}

// UpdateRemarkRequest 更新备注请求
type UpdateRemarkRequest struct {
	Remark string `json:"remark" binding:"max=50" example:"老朋友"`
}

// AddToBlacklistRequest 添加黑名单请求
type AddToBlacklistRequest struct {
	UserId string `json:"userId" binding:"required" example:"user-456"`
}

// FriendResponse 好友信息响应
type FriendResponse struct {
	UserID    string    `json:"userId" example:"user-123"`
	Remark    string    `json:"remark" example:"老朋友"`
	CreatedAt time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
	UserInfo  *UserInfo `json:"userInfo,omitempty"`
}

// FriendListResponse 好友列表响应
type FriendListResponse struct {
	Friends []*FriendResponse `json:"friends"`
	Total   int64             `json:"total" example:"10"`
}

// FriendRequestResponse 好友申请响应
type FriendRequestResponse struct {
	ID           int64     `json:"id" example:"1"`
	FromUserID   string    `json:"fromUserId" example:"user-123"`
	ToUserID     string    `json:"toUserId" example:"user-456"`
	Message      string    `json:"message" example:"你好"`
	Source       string    `json:"source" example:"search"`
	Status       string    `json:"status" example:"pending"`
	CreatedAt    time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	FromUserInfo *UserInfo `json:"fromUserInfo,omitempty"`
}

// FriendRequestListResponse 好友申请列表响应
type FriendRequestListResponse struct {
	Requests []*FriendRequestResponse `json:"requests"`
	Total    int64                    `json:"total" example:"5"`
}

// SendFriendRequestResponse 发送好友申请响应
type SendFriendRequestResponse struct {
	RequestID    int64 `json:"requestId" example:"1"`
	AutoAccepted bool  `json:"autoAccepted" example:"false"`
}

// BlacklistItemResponse 黑名单项响应
type BlacklistItemResponse struct {
	ID              int64     `json:"id" example:"1"`
	UserID          string    `json:"userId" example:"user-123"`
	BlockedUserID   string    `json:"blockedUserId" example:"user-456"`
	CreatedAt       time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	BlockedUserInfo *UserInfo `json:"blockedUserInfo,omitempty"`
}

// BlacklistResponse 黑名单列表响应
type BlacklistResponse struct {
	Items []*BlacklistItemResponse `json:"items"`
	Total int64                    `json:"total" example:"2"`
}

// UserInfo 用户基本信息（从user-service获取）
type UserInfo struct {
	UserID   string  `json:"userId" example:"user-123"`
	Nickname string  `json:"nickname" example:"张三"`
	Avatar   string  `json:"avatar" example:"https://example.com/avatar.jpg"`
	Gender   *int32  `json:"gender,omitempty" example:"1"`
	Bio      *string `json:"bio,omitempty" example:"个性签名"`
}
