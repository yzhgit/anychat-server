package dto

import "time"

// UpdateProfileRequest 更新个人资料请求
type UpdateProfileRequest struct {
	Nickname  *string    `json:"nickname"`
	Avatar    *string    `json:"avatar"`
	Signature *string    `json:"signature"`
	Gender    *int       `json:"gender"`
	Birthday  *time.Time `json:"birthday"`
	Region    *string    `json:"region"`
}

// UpdateSettingsRequest 更新个人设置请求
type UpdateSettingsRequest struct {
	NotificationEnabled   *bool   `json:"notificationEnabled"`
	SoundEnabled          *bool   `json:"soundEnabled"`
	VibrationEnabled      *bool   `json:"vibrationEnabled"`
	MessagePreviewEnabled *bool   `json:"messagePreviewEnabled"`
	FriendVerifyRequired  *bool   `json:"friendVerifyRequired"`
	SearchByPhone         *bool   `json:"searchByPhone"`
	SearchByID            *bool   `json:"searchById"`
	Language              *string `json:"language"`
}

// UpdatePushTokenRequest 更新推送Token请求
type UpdatePushTokenRequest struct {
	DeviceID  string `json:"deviceId" binding:"required"`
	PushToken string `json:"pushToken" binding:"required"`
	Platform  string `json:"platform" binding:"required"` // iOS/Android
}

// SearchUsersRequest 搜索用户请求
type SearchUsersRequest struct {
	Keyword  string `form:"keyword" binding:"required"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}
