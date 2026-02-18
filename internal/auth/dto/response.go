package dto

// RegisterResponse 注册响应
type RegisterResponse struct {
	UserID       string `json:"userId"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"` // 秒
}

// LoginResponse 登录响应
type LoginResponse struct {
	UserID       string       `json:"userId"`
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	ExpiresIn    int64        `json:"expiresIn"` // 秒
	User         *UserInfo    `json:"user"`
}

// UserInfo 用户信息
type UserInfo struct {
	UserID   string  `json:"userId"`
	Nickname string  `json:"nickname"`
	Avatar   string  `json:"avatar"`
	Phone    *string `json:"phone,omitempty"`
	Email    *string `json:"email,omitempty"`
}

// RefreshTokenResponse 刷新Token响应
type RefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"` // 秒
}

// TokenInfo Token信息
type TokenInfo struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}
