package dto

import "time"

// UserProfileResponse 用户资料响应
type UserProfileResponse struct {
	UserID     string     `json:"userId"`
	Nickname   string     `json:"nickname"`
	Avatar     string     `json:"avatar"`
	Signature  string     `json:"signature"`
	Gender     int        `json:"gender"`
	Birthday   *time.Time `json:"birthday,omitempty"`
	Region     string     `json:"region"`
	Phone      *string    `json:"phone,omitempty"`
	Email      *string    `json:"email,omitempty"`
	QRCodeURL  string     `json:"qrcodeUrl"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// UserInfoResponse 用户信息响应（查询其他用户时）
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

// UserSettingsResponse 用户设置响应
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

// QRCodeResponse 二维码响应
type QRCodeResponse struct {
	QRCodeURL string    `json:"qrcodeUrl"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// SearchUsersResponse 搜索用户响应
type SearchUsersResponse struct {
	Total int64              `json:"total"`
	Users []*UserBriefInfo   `json:"users"`
}

// UserBriefInfo 用户简要信息
type UserBriefInfo struct {
	UserID    string `json:"userId"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Signature string `json:"signature"`
}
