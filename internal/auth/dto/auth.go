package dto

// SendVerificationCodeRequest 发送验证码请求
type SendVerificationCodeRequest struct {
	Target     string `json:"target" binding:"required"`
	TargetType string `json:"targetType" binding:"required"`
	Purpose    string `json:"purpose" binding:"required"`
	DeviceID   string `json:"deviceId"`
	IPAddress  string `json:"ipAddress"`
}

// SendVerificationCodeResponse 发送验证码响应
type SendVerificationCodeResponse struct {
	CodeID    string `json:"codeId"`
	ExpiresIn int64  `json:"expiresIn"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	PhoneNumber   string `json:"phoneNumber" binding:"required_without=Email"`
	Email         string `json:"email" binding:"required_without=PhoneNumber,omitempty,email"`
	Password      string `json:"password" binding:"required,min=8,max=32"`
	VerifyCode    string `json:"verifyCode" binding:"required"`
	Nickname      string `json:"nickname"`
	DeviceType    string `json:"deviceType" binding:"required"`
	DeviceID      string `json:"deviceId" binding:"required"`
	ClientVersion string `json:"clientVersion" binding:"required"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	UserID       string `json:"userId"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"` // 秒
}

// LoginRequest 登录请求
type LoginRequest struct {
	Account       string `json:"account" binding:"required"`
	Password      string `json:"password" binding:"required"`
	DeviceType    string `json:"deviceType" binding:"required"`
	DeviceID      string `json:"deviceId" binding:"required"`
	ClientVersion string `json:"clientVersion" binding:"required"`
	IpAddress     string `json:"ipAddress"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	UserID       string    `json:"userId"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresIn    int64     `json:"expiresIn"` // 秒
	User         *UserInfo `json:"user"`
}

// UserInfo 用户信息
type UserInfo struct {
	UserID   string  `json:"userId"`
	Nickname string  `json:"nickname"`
	Avatar   string  `json:"avatar"`
	Phone    *string `json:"phone,omitempty"`
	Email    *string `json:"email,omitempty"`
}

// RefreshTokenRequest 刷新Token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// RefreshTokenResponse 刷新Token响应
type RefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"` // 秒
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=32"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Account     string `json:"account" binding:"required"`
	VerifyCode  string `json:"verifyCode" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=32"`
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	DeviceID string `json:"deviceId" binding:"required"`
}

// TokenInfo Token信息
type TokenInfo struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}
