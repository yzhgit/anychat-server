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

// BindPhoneRequest 绑定手机号请求
type BindPhoneRequest struct {
	PhoneNumber string `json:"phoneNumber" binding:"required"`
	VerifyCode  string `json:"verifyCode" binding:"required"`
}

// ChangePhoneRequest 更换手机号请求
type ChangePhoneRequest struct {
	OldPhoneNumber string  `json:"oldPhoneNumber" binding:"required"`
	NewPhoneNumber string  `json:"newPhoneNumber" binding:"required"`
	NewVerifyCode  string  `json:"newVerifyCode" binding:"required"`
	OldVerifyCode  *string `json:"oldVerifyCode"`
	DeviceID       string  `json:"-"`
}

// BindEmailRequest 绑定邮箱请求
type BindEmailRequest struct {
	Email      string `json:"email" binding:"required"`
	VerifyCode string `json:"verifyCode" binding:"required"`
}

// ChangeEmailRequest 更换邮箱请求
type ChangeEmailRequest struct {
	OldEmail      string  `json:"oldEmail" binding:"required"`
	NewEmail      string  `json:"newEmail" binding:"required"`
	NewVerifyCode string  `json:"newVerifyCode" binding:"required"`
	OldVerifyCode *string `json:"oldVerifyCode"`
	DeviceID      string  `json:"-"`
}

// SearchUsersRequest 搜索用户请求
type SearchUsersRequest struct {
	Keyword  string `form:"keyword" binding:"required"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}
