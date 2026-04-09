package dto

// SendVerificationCodeRequest send verification code request
type SendVerificationCodeRequest struct {
	Target     string `json:"target" binding:"required"`
	TargetType string `json:"targetType" binding:"required"`
	Purpose    string `json:"purpose" binding:"required"`
	DeviceID   string `json:"deviceId"`
	IPAddress  string `json:"ipAddress"`
}

// SendVerificationCodeResponse send verification code response
type SendVerificationCodeResponse struct {
	CodeID    string `json:"codeId"`
	ExpiresIn int64  `json:"expiresIn"`
}

// RegisterRequest register request
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

// RegisterResponse register response
type RegisterResponse struct {
	UserID       string `json:"userId"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"` // seconds
}

// LoginRequest login request
type LoginRequest struct {
	Account       string `json:"account" binding:"required"`
	Password      string `json:"password" binding:"required"`
	DeviceType    string `json:"deviceType" binding:"required"`
	DeviceID      string `json:"deviceId" binding:"required"`
	ClientVersion string `json:"clientVersion" binding:"required"`
	IpAddress     string `json:"ipAddress"`
}

// LoginResponse login response
type LoginResponse struct {
	UserID       string    `json:"userId"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresIn    int64     `json:"expiresIn"` // seconds
	User         *UserInfo `json:"user"`
}

// UserInfo user info
type UserInfo struct {
	UserID   string  `json:"userId"`
	Nickname string  `json:"nickname"`
	Avatar   string  `json:"avatar"`
	Phone    *string `json:"phone,omitempty"`
	Email    *string `json:"email,omitempty"`
}

// RefreshTokenRequest refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// RefreshTokenResponse refresh token response
type RefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"` // seconds
}

// ChangePasswordRequest change password request
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=32"`
	DeviceID    string `json:"deviceId" binding:"required"`
}

// ResetPasswordRequest reset password request
type ResetPasswordRequest struct {
	Account     string `json:"account" binding:"required"`
	VerifyCode  string `json:"verifyCode" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=32"`
}

// LogoutRequest logout request
type LogoutRequest struct {
	DeviceID string `json:"deviceId" binding:"required"`
}

// TokenInfo token info
type TokenInfo struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}
