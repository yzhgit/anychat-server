package dto

import "time"

// UpdateProfileRequest update profile request
type UpdateProfileRequest struct {
	Nickname  *string    `json:"nickname"`
	Avatar    *string    `json:"avatar"`
	Signature *string    `json:"signature"`
	Gender    *int       `json:"gender"`
	Birthday  *time.Time `json:"birthday"`
	Region    *string    `json:"region"`
}

// UpdateSettingsRequest update settings request
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

// UpdatePushTokenRequest update push token request
type UpdatePushTokenRequest struct {
	DeviceID  string `json:"deviceId" binding:"required"`
	PushToken string `json:"pushToken" binding:"required"`
	Platform  string `json:"platform" binding:"required"` // iOS/Android
}

// BindPhoneRequest bind phone request
type BindPhoneRequest struct {
	PhoneNumber string `json:"phoneNumber" binding:"required"`
	VerifyCode  string `json:"verifyCode" binding:"required"`
}

// ChangePhoneRequest change phone request
type ChangePhoneRequest struct {
	OldPhoneNumber string  `json:"oldPhoneNumber" binding:"required"`
	NewPhoneNumber string  `json:"newPhoneNumber" binding:"required"`
	NewVerifyCode  string  `json:"newVerifyCode" binding:"required"`
	OldVerifyCode  *string `json:"oldVerifyCode"`
	DeviceID       string  `json:"-"`
}

// BindEmailRequest bind email request
type BindEmailRequest struct {
	Email      string `json:"email" binding:"required"`
	VerifyCode string `json:"verifyCode" binding:"required"`
}

// ChangeEmailRequest change email request
type ChangeEmailRequest struct {
	OldEmail      string  `json:"oldEmail" binding:"required"`
	NewEmail      string  `json:"newEmail" binding:"required"`
	NewVerifyCode string  `json:"newVerifyCode" binding:"required"`
	OldVerifyCode *string `json:"oldVerifyCode"`
	DeviceID      string  `json:"-"`
}

// SearchUsersRequest search users request
type SearchUsersRequest struct {
	Keyword  string `form:"keyword" binding:"required"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}
