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

// AuthHandler auth HTTP处理器
type AuthHandler struct {
	clientManager *client.Manager
}

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	PhoneNumber *string `json:"phoneNumber" example:"13800138000"`
	Email       *string `json:"email" example:"user@example.com"`
	Password    string  `json:"password" binding:"required" example:"password123"`
	VerifyCode  string  `json:"verifyCode" binding:"required" example:"123456"`
	Nickname    *string `json:"nickname" example:"张三"`
	DeviceType  string  `json:"deviceType" binding:"required" example:"ios" enums:"ios,android,web"`
	DeviceID    string  `json:"deviceId" binding:"required" example:"device-uuid-123"`
}

// LoginRequest 用户登录请求
type LoginRequest struct {
	Account    string `json:"account" binding:"required" example:"13800138000"`
	Password   string `json:"password" binding:"required" example:"password123"`
	DeviceType string `json:"deviceType" binding:"required" example:"ios" enums:"ios,android,web"`
	DeviceID   string `json:"deviceId" binding:"required" example:"device-uuid-123"`
}

// RefreshTokenRequest 刷新令牌请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	DeviceID string `json:"deviceId" binding:"required" example:"device-uuid-123"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required" example:"oldpass123"`
	NewPassword string `json:"newPassword" binding:"required" example:"newpass123"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	UserID       string      `json:"userId" example:"user-123"`
	AccessToken  string      `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string      `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresIn    int64       `json:"expiresIn" example:"7200"`
	User         *UserInfo   `json:"user,omitempty"`
}

// UserInfo 用户信息
type UserInfo struct {
	UserID   string  `json:"userId" example:"user-123"`
	Nickname string  `json:"nickname" example:"张三"`
	Avatar   string  `json:"avatar" example:"https://example.com/avatar.jpg"`
	Phone    *string `json:"phone,omitempty" example:"13800138000"`
	Email    *string `json:"email,omitempty" example:"user@example.com"`
}

// NewAuthHandler 创建auth处理器
func NewAuthHandler(clientManager *client.Manager) *AuthHandler {
	return &AuthHandler{
		clientManager: clientManager,
	}
}

// Register 用户注册
// @Summary      用户注册
// @Description  用户通过手机号或邮箱注册新账号
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "注册信息"
// @Success      200      {object}  response.Response{data=AuthResponse}  "注册成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      409      {object}  response.Response  "用户已存在"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	// 调用auth-service gRPC
	resp, err := h.clientManager.Auth().Register(c.Request.Context(), &authpb.RegisterRequest{
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
		Password:    req.Password,
		VerifyCode:  req.VerifyCode,
		Nickname:    req.Nickname,
		DeviceType:  req.DeviceType,
		DeviceId:    req.DeviceID,
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

// Login 用户登录
// @Summary      用户登录
// @Description  用户通过账号密码登录
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "登录信息"
// @Success      200      {object}  response.Response{data=AuthResponse}  "登录成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "账号或密码错误"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	// 调用auth-service gRPC
	resp, err := h.clientManager.Auth().Login(c.Request.Context(), &authpb.LoginRequest{
		Account:    req.Account,
		Password:   req.Password,
		DeviceType: req.DeviceType,
		DeviceId:   req.DeviceID,
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

// Logout 用户登出
// @Summary      用户登出
// @Description  用户登出，使当前设备的令牌失效
// @Tags         认证
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      LogoutRequest  true  "登出信息"
// @Success      200      {object}  response.Response  "登出成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	userID := gwmiddleware.GetUserID(c)

	// 调用auth-service gRPC
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

// RefreshToken 刷新Token
// @Summary      刷新访问令牌
// @Description  使用刷新令牌获取新的访问令牌
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      RefreshTokenRequest  true  "刷新令牌"
// @Success      200      {object}  response.Response{data=AuthResponse}  "刷新成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "刷新令牌无效"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	// 调用auth-service gRPC
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

// ChangePassword 修改密码
// @Summary      修改密码
// @Description  用户修改登录密码
// @Tags         认证
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      ChangePasswordRequest  true  "修改密码信息"
// @Success      200      {object}  response.Response  "修改成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权或原密码错误"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /auth/password/change [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	userID := gwmiddleware.GetUserID(c)

	// 调用auth-service gRPC
	_, err := h.clientManager.Auth().ChangePassword(c.Request.Context(), &authpb.ChangePasswordRequest{
		UserId:      userID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// handleGRPCError 处理gRPC错误
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
	default:
		response.InternalError(c, st.Message())
	}
}
