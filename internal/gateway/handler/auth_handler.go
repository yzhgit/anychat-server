package handler

import (
	"github.com/anychat/server/api/proto/auth"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthHandler auth HTTP handler
type AuthHandler struct {
	clientManager *client.Manager
}

// SendCodeRequest send verification code request
type SendCodeRequest struct {
	Target     string `json:"target" binding:"required" example:"13800138000"`
	TargetType string `json:"targetType" binding:"required" example:"sms" enums:"sms,email"`
	Purpose    string `json:"purpose" binding:"required" example:"register"`
	DeviceID   string `json:"deviceId" example:"device-uuid-123"`
}

// RegisterRequest user registration request
type RegisterRequest struct {
	PhoneNumber   *string `json:"phoneNumber" example:"13800138000"`
	Email         *string `json:"email" example:"user@example.com"`
	Password      string  `json:"password" binding:"required" example:"password123"`
	VerifyCode    string  `json:"verifyCode" binding:"required" example:"123456"`
	Nickname      *string `json:"nickname" example:"张三"`
	DeviceType    string  `json:"deviceType" binding:"required" example:"ios" enums:"ios,android,web"`
	DeviceID      string  `json:"deviceId" binding:"required" example:"device-uuid-123"`
	ClientVersion string  `json:"clientVersion" binding:"required" example:"1.0.0"`
}

// LoginRequest user login request
type LoginRequest struct {
	Account       string `json:"account" binding:"required" example:"13800138000"`
	Password      string `json:"password" binding:"required" example:"password123"`
	DeviceType    string `json:"deviceType" binding:"required" example:"ios" enums:"ios,android,web"`
	DeviceID      string `json:"deviceId" binding:"required" example:"device-uuid-123"`
	ClientVersion string `json:"clientVersion" binding:"required" example:"1.0.0"`
	IpAddress     string `json:"ipAddress"`
}

// RefreshTokenRequest refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// LogoutRequest logout request
type LogoutRequest struct {
	DeviceID string `json:"deviceId" binding:"required" example:"device-uuid-123"`
}

// ChangePasswordRequest change password request
type ChangePasswordRequest struct {
	DeviceID    string `json:"deviceId" binding:"required" example:"device-uuid-123"`
	OldPassword string `json:"oldPassword" binding:"required" example:"oldpass123"`
	NewPassword string `json:"newPassword" binding:"required" example:"newpass123"`
}

// ResetPasswordRequest reset password request (forgot password)
type ResetPasswordRequest struct {
	Account     string `json:"account" binding:"required" example:"13800138000"`
	VerifyCode  string `json:"verifyCode" binding:"required" example:"123456"`
	NewPassword string `json:"newPassword" binding:"required" example:"NewPass123"`
}

// AuthResponse auth response
type AuthResponse struct {
	UserID       string    `json:"userId" example:"user-123"`
	AccessToken  string    `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string    `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresIn    int64     `json:"expiresIn" example:"7200"`
	User         *UserInfo `json:"user,omitempty"`
}

// SendCodeResponse send verification code response
type SendCodeResponse struct {
	CodeID    string `json:"codeId" example:"vc_20260405_xxx"`
	ExpiresIn int64  `json:"expiresIn" example:"300"`
}

// UserInfo user info
type UserInfo struct {
	UserID   string  `json:"userId" example:"user-123"`
	Nickname string  `json:"nickname" example:"张三"`
	Avatar   string  `json:"avatar" example:"https://example.com/avatar.jpg"`
	Phone    *string `json:"phone,omitempty" example:"13800138000"`
	Email    *string `json:"email,omitempty" example:"user@example.com"`
}

// NewAuthHandler creates auth handler
func NewAuthHandler(clientManager *client.Manager) *AuthHandler {
	return &AuthHandler{
		clientManager: clientManager,
	}
}

// SendCode send verification code
// @Summary      send verification code
// @Description  Send SMS/email verification code for registration, password recovery or binding scenarios
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      SendCodeRequest  true  "verification code request"
// @Success      200      {object}  response.Response{data=SendCodeResponse}  "send success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      429      {object}  response.Response  "too many requests"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /auth/send-code [post]
func (h *AuthHandler) SendCode(c *gin.Context) {
	var req SendCodeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.clientManager.Auth().SendVerificationCode(c.Request.Context(), &authpb.SendVerificationCodeRequest{
		Target:     req.Target,
		TargetType: req.TargetType,
		Purpose:    req.Purpose,
		DeviceId:   req.DeviceID,
		IpAddress:  c.ClientIP(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"codeId":    resp.CodeId,
		"expiresIn": resp.ExpiresIn,
	})
}

// Register user registration
// @Summary      user registration
// @Description  User registers new account via phone or email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "registration info"
// @Success      200      {object}  response.Response{data=AuthResponse}  "registration success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      409      {object}  response.Response  "user already exists"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	// Call auth-service gRPC
	resp, err := h.clientManager.Auth().Register(c.Request.Context(), &authpb.RegisterRequest{
		PhoneNumber:   req.PhoneNumber,
		Email:         req.Email,
		Password:      req.Password,
		VerifyCode:    req.VerifyCode,
		Nickname:      req.Nickname,
		DeviceType:    req.DeviceType,
		DeviceId:      req.DeviceID,
		ClientVersion: req.ClientVersion,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"userId":       resp.UserId,
		"accessToken":  resp.AccessToken,
		"refreshToken": resp.RefreshToken,
		"expiresIn":    resp.ExpiresIn,
	})
}

// Login user login
// @Summary      user login
// @Description  User login via account and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "login info"
// @Success      200      {object}  response.Response{data=AuthResponse}  "login success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "incorrect account or password"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	// Call auth-service gRPC
	resp, err := h.clientManager.Auth().Login(c.Request.Context(), &authpb.LoginRequest{
		Account:       req.Account,
		Password:      req.Password,
		DeviceType:    req.DeviceType,
		DeviceId:      req.DeviceID,
		ClientVersion: req.ClientVersion,
		IpAddress:     c.ClientIP(),
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	result := gin.H{
		"userId":       resp.UserId,
		"accessToken":  resp.AccessToken,
		"refreshToken": resp.RefreshToken,
		"expiresIn":    resp.ExpiresIn,
	}

	if resp.User != nil {
		result["user"] = gin.H{
			"userId":   resp.User.UserId,
			"nickname": resp.User.Nickname,
			"avatar":   resp.User.Avatar,
			"phone":    resp.User.Phone,
			"email":    resp.User.Email,
		}
	}

	response.Success(c, result)
}

// Logout user logout
// @Summary      user logout
// @Description  User logout, invalidate token for current device
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      LogoutRequest  true  "logout info"
// @Success      200      {object}  response.Response  "logout success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	userID := gwmiddleware.GetUserID(c)

	// Call auth-service gRPC
	_, err := h.clientManager.Auth().Logout(c.Request.Context(), &authpb.LogoutRequest{
		UserId:   userID,
		DeviceId: req.DeviceID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// RefreshToken refresh access token
// @Summary      refresh access token
// @Description  Use refresh token to get new access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RefreshTokenRequest  true  "refresh token"
// @Success      200      {object}  response.Response{data=AuthResponse}  "refresh success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "invalid refresh token"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	// Call auth-service gRPC
	resp, err := h.clientManager.Auth().RefreshToken(c.Request.Context(), &authpb.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"accessToken":  resp.AccessToken,
		"refreshToken": resp.RefreshToken,
		"expiresIn":    resp.ExpiresIn,
	})
}

// ChangePassword change password
// @Summary      change password
// @Description  User changes login password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      ChangePasswordRequest  true  "change password info"
// @Success      200      {object}  response.Response  "change success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized or wrong original password"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /auth/password/change [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	userID := gwmiddleware.GetUserID(c)

	// Call auth-service gRPC
	_, err := h.clientManager.Auth().ChangePassword(c.Request.Context(), &authpb.ChangePasswordRequest{
		UserId:      userID,
		DeviceId:    req.DeviceID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// ResetPassword reset password
// @Summary      reset password
// @Description  User forgets password, reset via verification code
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      ResetPasswordRequest  true  "reset password info"
// @Success      200      {object}  response.Response  "reset success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "verification code error"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /auth/password/reset [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	// Call auth-service gRPC
	_, err := h.clientManager.Auth().ResetPassword(c.Request.Context(), &authpb.ResetPasswordRequest{
		Account:     req.Account,
		VerifyCode:  req.VerifyCode,
		NewPassword: req.NewPassword,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// handleGRPCError handle gRPC errors
func handleGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		response.InternalError(c, err.Error())
		return
	}

	switch st.Code() {
	case codes.InvalidArgument:
		response.ParamError(c, st.Message())
	case codes.NotFound:
		response.Error(c, 404, st.Message())
	case codes.AlreadyExists:
		response.Error(c, 409, st.Message())
	case codes.Unauthenticated:
		response.Error(c, 401, st.Message())
	case codes.PermissionDenied:
		response.Error(c, 403, st.Message())
	case codes.ResourceExhausted:
		response.Error(c, 429, st.Message())
	default:
		response.InternalError(c, st.Message())
	}
}
