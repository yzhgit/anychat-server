package handler

import (
	"strconv"
	"time"

	userpb "github.com/anychat/server/api/proto/user"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserHandler user HTTP处理器
type UserHandler struct {
	clientManager *client.Manager
}

// UpdateProfileRequest 更新资料请求
type UpdateProfileRequest struct {
	Nickname  *string    `json:"nickname" example:"张三"`
	Avatar    *string    `json:"avatar" example:"https://example.com/avatar.jpg"`
	Signature *string    `json:"signature" example:"这是我的个性签名"`
	Gender    *int32     `json:"gender" example:"1" enums:"0,1,2"`
	Birthday  *time.Time `json:"birthday" example:"1990-01-01T00:00:00Z"`
	Region    *string    `json:"region" example:"北京"`
}

// UserProfile 用户资料
type UserProfile struct {
	UserID    string     `json:"userId" example:"user-123"`
	Nickname  string     `json:"nickname" example:"张三"`
	Avatar    string     `json:"avatar" example:"https://example.com/avatar.jpg"`
	Signature string     `json:"signature" example:"这是我的个性签名"`
	Gender    int32      `json:"gender" example:"1"`
	Region    string     `json:"region" example:"北京"`
	QRCodeURL string     `json:"qrcodeUrl" example:"https://example.com/qrcode.png"`
	Birthday  *time.Time `json:"birthday,omitempty" example:"1990-01-01T00:00:00Z"`
	Phone     *string    `json:"phone,omitempty" example:"13800138000"`
	Email     *string    `json:"email,omitempty" example:"user@example.com"`
	CreatedAt time.Time  `json:"createdAt" example:"2024-01-01T00:00:00Z"`
}

// UserSearchResult 用户搜索结果
type UserSearchResult struct {
	Total int64            `json:"total" example:"100"`
	Users []UserSearchItem `json:"users"`
}

// UserSearchItem 搜索结果项
type UserSearchItem struct {
	UserID    string `json:"userId" example:"user-123"`
	Nickname  string `json:"nickname" example:"张三"`
	Avatar    string `json:"avatar" example:"https://example.com/avatar.jpg"`
	Signature string `json:"signature" example:"这是我的个性签名"`
}

// UserSettings 用户设置
type UserSettings struct {
	UserID                string `json:"userId" example:"user-123"`
	NotificationEnabled   bool   `json:"notificationEnabled" example:"true"`
	SoundEnabled          bool   `json:"soundEnabled" example:"true"`
	VibrationEnabled      bool   `json:"vibrationEnabled" example:"true"`
	MessagePreviewEnabled bool   `json:"messagePreviewEnabled" example:"true"`
	FriendVerifyRequired  bool   `json:"friendVerifyRequired" example:"true"`
	SearchByPhone         bool   `json:"searchByPhone" example:"true"`
	SearchByID            bool   `json:"searchById" example:"true"`
	Language              string `json:"language" example:"zh-CN"`
}

// UpdateSettingsRequest 更新设置请求
type UpdateSettingsRequest struct {
	NotificationEnabled   *bool   `json:"notificationEnabled" example:"true"`
	SoundEnabled          *bool   `json:"soundEnabled" example:"true"`
	VibrationEnabled      *bool   `json:"vibrationEnabled" example:"true"`
	MessagePreviewEnabled *bool   `json:"messagePreviewEnabled" example:"true"`
	FriendVerifyRequired  *bool   `json:"friendVerifyRequired" example:"true"`
	SearchByPhone         *bool   `json:"searchByPhone" example:"true"`
	SearchByID            *bool   `json:"searchById" example:"true"`
	Language              *string `json:"language" example:"zh-CN"`
}

// UpdatePushTokenRequest 更新推送Token请求
type UpdatePushTokenRequest struct {
	DeviceID  string `json:"deviceId" binding:"required" example:"device-uuid-123"`
	PushToken string `json:"pushToken" binding:"required" example:"push-token-xxx"`
	Platform  string `json:"platform" binding:"required" example:"ios" enums:"ios,android"`
}

// BindPhoneRequest 绑定手机号请求
type BindPhoneRequest struct {
	PhoneNumber string `json:"phoneNumber" binding:"required" example:"13800138000"`
	VerifyCode  string `json:"verifyCode" binding:"required" example:"123456"`
}

// ChangePhoneRequest 更换手机号请求
type ChangePhoneRequest struct {
	OldPhoneNumber string  `json:"oldPhoneNumber" binding:"required" example:"13800138000"`
	NewPhoneNumber string  `json:"newPhoneNumber" binding:"required" example:"13900139000"`
	NewVerifyCode  string  `json:"newVerifyCode" binding:"required" example:"123456"`
	OldVerifyCode  *string `json:"oldVerifyCode,omitempty" example:"123456"`
}

// BindEmailRequest 绑定邮箱请求
type BindEmailRequest struct {
	Email      string `json:"email" binding:"required" example:"user@example.com"`
	VerifyCode string `json:"verifyCode" binding:"required" example:"123456"`
}

// ChangeEmailRequest 更换邮箱请求
type ChangeEmailRequest struct {
	OldEmail      string  `json:"oldEmail" binding:"required" example:"old@example.com"`
	NewEmail      string  `json:"newEmail" binding:"required" example:"new@example.com"`
	NewVerifyCode string  `json:"newVerifyCode" binding:"required" example:"123456"`
	OldVerifyCode *string `json:"oldVerifyCode,omitempty" example:"123456"`
}

// NewUserHandler 创建user处理器
func NewUserHandler(clientManager *client.Manager) *UserHandler {
	return &UserHandler{
		clientManager: clientManager,
	}
}

// GetProfile 获取个人资料
// @Summary      获取个人资料
// @Description  获取当前登录用户的详细资料
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=UserProfile}  "获取成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /users/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	resp, err := h.clientManager.User().GetProfile(c.Request.Context(), &userpb.GetProfileRequest{
		UserId: userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	result := gin.H{
		"userId":    resp.UserId,
		"nickname":  resp.Nickname,
		"avatar":    resp.Avatar,
		"signature": resp.Signature,
		"gender":    resp.Gender,
		"region":    resp.Region,
		"qrcodeUrl": resp.QrcodeUrl,
		"createdAt": resp.CreatedAt.AsTime(),
	}

	if resp.Birthday != nil {
		result["birthday"] = resp.Birthday.AsTime()
	}
	if resp.Phone != nil {
		result["phone"] = *resp.Phone
	}
	if resp.Email != nil {
		result["email"] = *resp.Email
	}

	response.Success(c, result)
}

// UpdateProfile 更新个人资料
// @Summary      更新个人资料
// @Description  更新当前登录用户的个人资料
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      UpdateProfileRequest  true  "资料信息"
// @Success      200      {object}  response.Response{data=UserProfile}  "更新成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /users/me [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	userID := gwmiddleware.GetUserID(c)

	pbReq := &userpb.UpdateProfileRequest{
		UserId:    userID,
		Nickname:  req.Nickname,
		Avatar:    req.Avatar,
		Signature: req.Signature,
		Gender:    req.Gender,
		Region:    req.Region,
	}

	if req.Birthday != nil {
		pbReq.Birthday = timestamppb.New(*req.Birthday)
	}

	resp, err := h.clientManager.User().UpdateProfile(c.Request.Context(), pbReq)

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	result := gin.H{
		"userId":    resp.UserId,
		"nickname":  resp.Nickname,
		"avatar":    resp.Avatar,
		"signature": resp.Signature,
		"gender":    resp.Gender,
		"region":    resp.Region,
		"qrcodeUrl": resp.QrcodeUrl,
		"createdAt": resp.CreatedAt.AsTime(),
	}

	if resp.Birthday != nil {
		result["birthday"] = resp.Birthday.AsTime()
	}

	response.Success(c, result)
}

// GetUserInfo 获取用户信息
// @Summary      获取用户信息
// @Description  获取指定用户的公开信息
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId  path      string  true  "用户ID"
// @Success      200     {object}  response.Response{data=UserSearchItem}  "获取成功"
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      404     {object}  response.Response  "用户不存在"
// @Failure      500     {object}  response.Response  "服务器错误"
// @Router       /users/{userId} [get]
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	targetUserID := c.Param("userId")
	userID := gwmiddleware.GetUserID(c)

	resp, err := h.clientManager.User().GetUserInfo(c.Request.Context(), &userpb.GetUserInfoRequest{
		UserId:       userID,
		TargetUserId: targetUserID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"userId":    resp.UserId,
		"nickname":  resp.Nickname,
		"avatar":    resp.Avatar,
		"signature": resp.Signature,
		"gender":    resp.Gender,
		"region":    resp.Region,
		"isFriend":  resp.IsFriend,
		"isBlocked": resp.IsBlocked,
	})
}

// SearchUsers 搜索用户
// @Summary      搜索用户
// @Description  通过关键字搜索用户（支持昵称、手机号、用户ID）
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        keyword   query     string  true   "搜索关键字"
// @Param        page      query     int     false  "页码" default(1)
// @Param        pageSize  query     int     false  "每页数量" default(20)
// @Success      200       {object}  response.Response{data=UserSearchResult}  "搜索成功"
// @Failure      400       {object}  response.Response  "参数错误"
// @Failure      401       {object}  response.Response  "未授权"
// @Failure      500       {object}  response.Response  "服务器错误"
// @Router       /users/search [get]
func (h *UserHandler) SearchUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		response.ParamError(c, "keyword is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	resp, err := h.clientManager.User().SearchUsers(c.Request.Context(), &userpb.SearchUsersRequest{
		Keyword:  keyword,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	users := make([]gin.H, 0, len(resp.Users))
	for _, u := range resp.Users {
		users = append(users, gin.H{
			"userId":    u.UserId,
			"nickname":  u.Nickname,
			"avatar":    u.Avatar,
			"signature": u.Signature,
		})
	}

	response.Success(c, gin.H{
		"total": resp.Total,
		"users": users,
	})
}

// GetSettings 获取用户设置
// @Summary      获取用户设置
// @Description  获取当前用户的偏好设置
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=UserSettings}  "获取成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /users/me/settings [get]
func (h *UserHandler) GetSettings(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	resp, err := h.clientManager.User().GetSettings(c.Request.Context(), &userpb.GetSettingsRequest{
		UserId: userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"userId":                resp.UserId,
		"notificationEnabled":   resp.NotificationEnabled,
		"soundEnabled":          resp.SoundEnabled,
		"vibrationEnabled":      resp.VibrationEnabled,
		"messagePreviewEnabled": resp.MessagePreviewEnabled,
		"friendVerifyRequired":  resp.FriendVerifyRequired,
		"searchByPhone":         resp.SearchByPhone,
		"searchById":            resp.SearchById,
		"language":              resp.Language,
	})
}

// UpdateSettings 更新用户设置
// @Summary      更新用户设置
// @Description  更新当前用户的偏好设置
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      UpdateSettingsRequest  true  "设置信息"
// @Success      200      {object}  response.Response{data=UserSettings}  "更新成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /users/me/settings [put]
func (h *UserHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	userID := gwmiddleware.GetUserID(c)

	resp, err := h.clientManager.User().UpdateSettings(c.Request.Context(), &userpb.UpdateSettingsRequest{
		UserId:                userID,
		NotificationEnabled:   req.NotificationEnabled,
		SoundEnabled:          req.SoundEnabled,
		VibrationEnabled:      req.VibrationEnabled,
		MessagePreviewEnabled: req.MessagePreviewEnabled,
		FriendVerifyRequired:  req.FriendVerifyRequired,
		SearchByPhone:         req.SearchByPhone,
		SearchById:            req.SearchByID,
		Language:              req.Language,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"userId":                resp.UserId,
		"notificationEnabled":   resp.NotificationEnabled,
		"soundEnabled":          resp.SoundEnabled,
		"vibrationEnabled":      resp.VibrationEnabled,
		"messagePreviewEnabled": resp.MessagePreviewEnabled,
		"friendVerifyRequired":  resp.FriendVerifyRequired,
		"searchByPhone":         resp.SearchByPhone,
		"searchById":            resp.SearchById,
		"language":              resp.Language,
	})
}

// RefreshQRCode 刷新二维码
func (h *UserHandler) RefreshQRCode(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	resp, err := h.clientManager.User().RefreshQRCode(c.Request.Context(), &userpb.RefreshQRCodeRequest{
		UserId: userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"qrcodeUrl": resp.QrcodeUrl,
		"expiresAt": resp.ExpiresAt.AsTime(),
	})
}

// GetUserByQRCode 通过二维码获取用户
func (h *UserHandler) GetUserByQRCode(c *gin.Context) {
	qrcode := c.Query("qrcode")
	if qrcode == "" {
		response.ParamError(c, "qrcode is required")
		return
	}

	resp, err := h.clientManager.User().GetUserByQRCode(c.Request.Context(), &userpb.GetUserByQRCodeRequest{
		Qrcode: qrcode,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"userId":    resp.UserId,
		"nickname":  resp.Nickname,
		"avatar":    resp.Avatar,
		"signature": resp.Signature,
	})
}

// UpdatePushToken 更新推送Token
// @Summary      更新推送Token
// @Description  更新设备的推送通知Token
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      UpdatePushTokenRequest  true  "推送Token信息"
// @Success      200      {object}  response.Response  "更新成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /users/me/push-token [post]
func (h *UserHandler) UpdatePushToken(c *gin.Context) {
	var req UpdatePushTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	userID := gwmiddleware.GetUserID(c)

	_, err := h.clientManager.User().UpdatePushToken(c.Request.Context(), &userpb.UpdatePushTokenRequest{
		UserId:    userID,
		DeviceId:  req.DeviceID,
		PushToken: req.PushToken,
		Platform:  req.Platform,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// BindPhone 绑定手机号
// @Summary      绑定手机号
// @Description  为当前登录用户绑定手机号
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      BindPhoneRequest  true  "绑定手机号信息"
// @Success      200      {object}  response.Response{data=object}  "绑定成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      409      {object}  response.Response  "手机号已被占用"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /users/me/phone/bind [post]
func (h *UserHandler) BindPhone(c *gin.Context) {
	var req BindPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	userID := gwmiddleware.GetUserID(c)
	resp, err := h.clientManager.User().BindPhone(c.Request.Context(), &userpb.BindPhoneRequest{
		UserId:      userID,
		PhoneNumber: req.PhoneNumber,
		VerifyCode:  req.VerifyCode,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"phoneNumber": resp.PhoneNumber,
		"isPrimary":   resp.IsPrimary,
	})
}

// ChangePhone 更换手机号
// @Summary      更换手机号
// @Description  为当前登录用户更换已绑定手机号
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      ChangePhoneRequest  true  "更换手机号信息"
// @Success      200      {object}  response.Response{data=object}  "更换成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      409      {object}  response.Response  "手机号已被占用"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /users/me/phone/change [post]
func (h *UserHandler) ChangePhone(c *gin.Context) {
	var req ChangePhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	userID := gwmiddleware.GetUserID(c)
	deviceID := gwmiddleware.GetDeviceID(c)
	pbReq := &userpb.ChangePhoneRequest{
		UserId:         userID,
		OldPhoneNumber: req.OldPhoneNumber,
		NewPhoneNumber: req.NewPhoneNumber,
		NewVerifyCode:  req.NewVerifyCode,
		DeviceId:       deviceID,
	}
	if req.OldVerifyCode != nil {
		pbReq.OldVerifyCode = req.OldVerifyCode
	}

	resp, err := h.clientManager.User().ChangePhone(c.Request.Context(), pbReq)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"oldPhoneNumber": resp.OldPhoneNumber,
		"newPhoneNumber": resp.NewPhoneNumber,
	})
}

// BindEmail 绑定邮箱
// @Summary      绑定邮箱
// @Description  为当前登录用户绑定邮箱
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      BindEmailRequest  true  "绑定邮箱信息"
// @Success      200      {object}  response.Response{data=object}  "绑定成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      409      {object}  response.Response  "邮箱已被占用"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /users/me/email/bind [post]
func (h *UserHandler) BindEmail(c *gin.Context) {
	var req BindEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	userID := gwmiddleware.GetUserID(c)
	resp, err := h.clientManager.User().BindEmail(c.Request.Context(), &userpb.BindEmailRequest{
		UserId:     userID,
		Email:      req.Email,
		VerifyCode: req.VerifyCode,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"email":     resp.Email,
		"isPrimary": resp.IsPrimary,
	})
}

// ChangeEmail 更换邮箱
// @Summary      更换邮箱
// @Description  为当前登录用户更换已绑定邮箱
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      ChangeEmailRequest  true  "更换邮箱信息"
// @Success      200      {object}  response.Response{data=object}  "更换成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      409      {object}  response.Response  "邮箱已被占用"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /users/me/email/change [post]
func (h *UserHandler) ChangeEmail(c *gin.Context) {
	var req ChangeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	userID := gwmiddleware.GetUserID(c)
	deviceID := gwmiddleware.GetDeviceID(c)
	pbReq := &userpb.ChangeEmailRequest{
		UserId:        userID,
		OldEmail:      req.OldEmail,
		NewEmail:      req.NewEmail,
		NewVerifyCode: req.NewVerifyCode,
		DeviceId:      deviceID,
	}
	if req.OldVerifyCode != nil {
		pbReq.OldVerifyCode = req.OldVerifyCode
	}

	resp, err := h.clientManager.User().ChangeEmail(c.Request.Context(), pbReq)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"oldEmail": resp.OldEmail,
		"newEmail": resp.NewEmail,
	})
}
