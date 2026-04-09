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

// UserHandler user HTTP handler
type UserHandler struct {
	clientManager *client.Manager
}

// UpdateProfileRequest update profile request
type UpdateProfileRequest struct {
	Nickname  *string    `json:"nickname" example:"张三"`
	Avatar    *string    `json:"avatar" example:"https://example.com/avatar.jpg"`
	Signature *string    `json:"signature" example:"这是我的个性签名"`
	Gender    *int32     `json:"gender" example:"1" enums:"0,1,2"`
	Birthday  *time.Time `json:"birthday" example:"1990-01-01T00:00:00Z"`
	Region    *string    `json:"region" example:"北京"`
}

// UserProfile user profile
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

// UserSearchResult user search result
type UserSearchResult struct {
	Total int64            `json:"total" example:"100"`
	Users []UserSearchItem `json:"users"`
}

// UserSearchItem search result item
type UserSearchItem struct {
	UserID    string `json:"userId" example:"user-123"`
	Nickname  string `json:"nickname" example:"张三"`
	Avatar    string `json:"avatar" example:"https://example.com/avatar.jpg"`
	Signature string `json:"signature" example:"这是我的个性签名"`
}

// UserSettings user settings
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

// UpdateSettingsRequest update settings request
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

// UpdatePushTokenRequest update push token request
type UpdatePushTokenRequest struct {
	DeviceID  string `json:"deviceId" binding:"required" example:"device-uuid-123"`
	PushToken string `json:"pushToken" binding:"required" example:"push-token-xxx"`
	Platform  string `json:"platform" binding:"required" example:"ios" enums:"ios,android"`
}

// BindPhoneRequest bind phone request
type BindPhoneRequest struct {
	PhoneNumber string `json:"phoneNumber" binding:"required" example:"13800138000"`
	VerifyCode  string `json:"verifyCode" binding:"required" example:"123456"`
}

// ChangePhoneRequest change phone request
type ChangePhoneRequest struct {
	OldPhoneNumber string  `json:"oldPhoneNumber" binding:"required" example:"13800138000"`
	NewPhoneNumber string  `json:"newPhoneNumber" binding:"required" example:"13900139000"`
	NewVerifyCode  string  `json:"newVerifyCode" binding:"required" example:"123456"`
	OldVerifyCode  *string `json:"oldVerifyCode,omitempty" example:"123456"`
}

// BindEmailRequest bind email request
type BindEmailRequest struct {
	Email      string `json:"email" binding:"required" example:"user@example.com"`
	VerifyCode string `json:"verifyCode" binding:"required" example:"123456"`
}

// ChangeEmailRequest change email request
type ChangeEmailRequest struct {
	OldEmail      string  `json:"oldEmail" binding:"required" example:"old@example.com"`
	NewEmail      string  `json:"newEmail" binding:"required" example:"new@example.com"`
	NewVerifyCode string  `json:"newVerifyCode" binding:"required" example:"123456"`
	OldVerifyCode *string `json:"oldVerifyCode,omitempty" example:"123456"`
}

// NewUserHandler creates user handler
func NewUserHandler(clientManager *client.Manager) *UserHandler {
	return &UserHandler{
		clientManager: clientManager,
	}
}

// GetProfile get personal profile
// @Summary      get personal profile
// @Description  Get detailed profile of currently logged in user
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=UserProfile}  "get success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// UpdateProfile update personal profile
// @Summary      update personal profile
// @Description  Update personal profile of currently logged in user
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      UpdateProfileRequest  true  "profile info"
// @Success      200      {object}  response.Response{data=UserProfile}  "update success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      500      {object}  response.Response  "server error"
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

// GetUserInfo get user info
// @Summary      get user info
// @Description  Get public info of specified user
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId  path      string  true  "user ID"
// @Success      200     {object}  response.Response{data=UserSearchItem}  "get success"
// @Failure      401     {object}  response.Response  "unauthorized"
// @Failure      404     {object}  response.Response  "user not found"
// @Failure      500     {object}  response.Response  "server error"
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

// SearchUsers search users
// @Summary      search users
// @Description  Search users by keyword (supports nickname, phone, user ID)
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        keyword   query     string  true   "search keyword"
// @Param        page      query     int     false  "page number" default(1)
// @Param        pageSize  query     int     false  "page size" default(20)
// @Success      200       {object}  response.Response{data=UserSearchResult}  "search success"
// @Failure      400       {object}  response.Response  "parameter error"
// @Failure      401       {object}  response.Response  "unauthorized"
// @Failure      500       {object}  response.Response  "server error"
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

// GetSettings get user settings
// @Summary      get user settings
// @Description  Get current user's preference settings
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=UserSettings}  "get success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// UpdateSettings update user settings
// @Summary      update user settings
// @Description  Update current user's preference settings
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      UpdateSettingsRequest  true  "settings info"
// @Success      200      {object}  response.Response{data=UserSettings}  "update success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      500      {object}  response.Response  "server error"
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

// RefreshQRCode refresh QR code
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

// GetUserByQRCode get user via QR code
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

// UpdatePushToken update push token
// @Summary      update push token
// @Description  Update device push notification token
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      UpdatePushTokenRequest  true  "push token info"
// @Success      200      {object}  response.Response  "update success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      500      {object}  response.Response  "server error"
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

// BindPhone bind phone
// @Summary      bind phone
// @Description  Bind phone number for currently logged in user
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      BindPhoneRequest  true  "bind phone info"
// @Success      200      {object}  response.Response{data=object}  "bind success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      409      {object}  response.Response  "phone number already in use"
// @Failure      500      {object}  response.Response  "server error"
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

// ChangePhone change phone
// @Summary      change phone
// @Description  Change bound phone number for currently logged in user
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      ChangePhoneRequest  true  "change phone info"
// @Success      200      {object}  response.Response{data=object}  "change success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      409      {object}  response.Response  "phone number already in use"
// @Failure      500      {object}  response.Response  "server error"
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

// BindEmail bind email
// @Summary      bind email
// @Description  Bind email for currently logged in user
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      BindEmailRequest  true  "bind email info"
// @Success      200      {object}  response.Response{data=object}  "bind success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      409      {object}  response.Response  "email already in use"
// @Failure      500      {object}  response.Response  "server error"
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

// ChangeEmail change email
// @Summary      change email
// @Description  Change bound email for currently logged in user
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      ChangeEmailRequest  true  "change email info"
// @Success      200      {object}  response.Response{data=object}  "change success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      409      {object}  response.Response  "email already in use"
// @Failure      500      {object}  response.Response  "server error"
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
