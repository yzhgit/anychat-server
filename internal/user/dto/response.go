package dto

import "time"

// UserProfileResponse user profile response
type UserProfileResponse struct {
	UserID    string     `json:"userId"`
	Nickname  string     `json:"nickname"`
	Avatar    string     `json:"avatar"`
	Signature string     `json:"signature"`
	Gender    int        `json:"gender"`
	Birthday  *time.Time `json:"birthday,omitempty"`
	Region    string     `json:"region"`
	Phone     *string    `json:"phone,omitempty"`
	Email     *string    `json:"email,omitempty"`
	QRCodeURL string     `json:"qrcodeUrl"`
	CreatedAt time.Time  `json:"createdAt"`
}

// UserInfoResponse user info response (when querying other users)
type UserInfoResponse struct {
	UserID    string `json:"userId"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Signature string `json:"signature"`
	Gender    int    `json:"gender"`
	Region    string `json:"region"`
	IsFriend  bool   `json:"isFriend"`
	IsBlocked bool   `json:"isBlocked"`
}

// UserSettingsResponse user settings response
type UserSettingsResponse struct {
	UserID                string `json:"userId"`
	NotificationEnabled   bool   `json:"notificationEnabled"`
	SoundEnabled          bool   `json:"soundEnabled"`
	VibrationEnabled      bool   `json:"vibrationEnabled"`
	MessagePreviewEnabled bool   `json:"messagePreviewEnabled"`
	FriendVerifyRequired  bool   `json:"friendVerifyRequired"`
	SearchByPhone         bool   `json:"searchByPhone"`
	SearchByID            bool   `json:"searchById"`
	Language              string `json:"language"`
}

// QRCodeResponse QR code response
type QRCodeResponse struct {
	QRCodeURL string    `json:"qrcodeUrl"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// SearchUsersResponse search users response
type SearchUsersResponse struct {
	Total int64            `json:"total"`
	Users []*UserBriefInfo `json:"users"`
}

// UserBriefInfo user brief info
type UserBriefInfo struct {
	UserID    string `json:"userId"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Signature string `json:"signature"`
}

// BindPhoneResponse bind phone response
type BindPhoneResponse struct {
	PhoneNumber string `json:"phoneNumber"`
	IsPrimary   bool   `json:"isPrimary"`
}

// ChangePhoneResponse change phone response
type ChangePhoneResponse struct {
	OldPhoneNumber string `json:"oldPhoneNumber"`
	NewPhoneNumber string `json:"newPhoneNumber"`
}

// BindEmailResponse bind email response
type BindEmailResponse struct {
	Email     string `json:"email"`
	IsPrimary bool   `json:"isPrimary"`
}

// ChangeEmailResponse change email response
type ChangeEmailResponse struct {
	OldEmail string `json:"oldEmail"`
	NewEmail string `json:"newEmail"`
}
