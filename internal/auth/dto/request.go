package dto

// RegisterRequest 注册请求
type RegisterRequest struct {
	PhoneNumber string `json:"phoneNumber" binding:"required_without=Email"`
	Email       string `json:"email" binding:"required_without=PhoneNumber,omitempty,email"`
	Password    string `json:"password" binding:"required,min=8,max=32"`
	VerifyCode  string `json:"verifyCode" binding:"required"`
	Nickname    string `json:"nickname"`
	DeviceType  string `json:"deviceType" binding:"required"`
	DeviceID    string `json:"deviceId" binding:"required"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Account    string `json:"account" binding:"required"`
	Password   string `json:"password" binding:"required"`
	DeviceType string `json:"deviceType" binding:"required"`
	DeviceID   string `json:"deviceId" binding:"required"`
}

// RefreshTokenRequest 刷新Token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
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
