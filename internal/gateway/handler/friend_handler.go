package handler

import (
	"strconv"

	friendpb "github.com/anychat/server/api/proto/friend"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// FriendHandler friend HTTP处理器
type FriendHandler struct {
	clientManager *client.Manager
}

// NewFriendHandler 创建friend处理器
func NewFriendHandler(clientManager *client.Manager) *FriendHandler {
	return &FriendHandler{
		clientManager: clientManager,
	}
}

// GetFriends 获取好友列表
// @Summary      获取好友列表
// @Description  获取当前用户的所有好友,支持增量同步
// @Tags         好友
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        lastUpdateTime  query  int64  false  "上次更新时间戳(增量同步)"
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /friends [get]
func (h *FriendHandler) GetFriends(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	// 解析查询参数
	var lastUpdateTime *int64
	if timeStr := c.Query("lastUpdateTime"); timeStr != "" {
		if t, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
			lastUpdateTime = &t
		}
	}

	resp, err := h.clientManager.Friend().GetFriendList(c.Request.Context(), &friendpb.GetFriendListRequest{
		UserId:         userID,
		LastUpdateTime: lastUpdateTime,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// SendFriendRequest 发送好友申请
// @Summary      发送好友申请
// @Description  向指定用户发送好友申请
// @Tags         好友
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  object  true  "申请信息"
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /friends/requests [post]
func (h *FriendHandler) SendFriendRequest(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req struct {
		UserID  string `json:"userId" binding:"required" example:"user-456"`
		Message string `json:"message" example:"你好,我想加你为好友"`
		Source  string `json:"source" binding:"required,oneof=search qrcode group contacts" example:"search"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.clientManager.Friend().SendFriendRequest(c.Request.Context(), &friendpb.SendFriendRequestRequest{
		FromUserId: userID,
		ToUserId:   req.UserID,
		Message:    req.Message,
		Source:     req.Source,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// HandleFriendRequest 处理好友申请
// @Summary      处理好友申请
// @Description  接受或拒绝好友申请
// @Tags         好友
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  int  true  "申请ID"
// @Param        request  body  object  true  "处理动作"
// @Success      200  {object}  response.Response  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /friends/requests/{id} [put]
func (h *FriendHandler) HandleFriendRequest(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.ParamError(c, "invalid request id")
		return
	}

	var req struct {
		Action string `json:"action" binding:"required,oneof=accept reject" example:"accept"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err = h.clientManager.Friend().HandleFriendRequest(c.Request.Context(), &friendpb.HandleFriendRequestRequest{
		UserId:    userID,
		RequestId: requestID,
		Action:    req.Action,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// GetFriendRequests 获取好友申请列表
// @Summary      获取好友申请列表
// @Description  获取收到的或发送的好友申请列表
// @Tags         好友
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        type  query  string  false  "类型: received(收到的)/sent(发送的)"  default(received)
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /friends/requests [get]
func (h *FriendHandler) GetFriendRequests(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	requestType := c.DefaultQuery("type", "received")

	resp, err := h.clientManager.Friend().GetFriendRequests(c.Request.Context(), &friendpb.GetFriendRequestsRequest{
		UserId: userID,
		Type:   requestType,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// DeleteFriend 删除好友
// @Summary      删除好友
// @Description  删除指定好友
// @Tags         好友
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "好友用户ID"
// @Success      200  {object}  response.Response  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /friends/{id} [delete]
func (h *FriendHandler) DeleteFriend(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	friendID := c.Param("id")

	_, err := h.clientManager.Friend().DeleteFriend(c.Request.Context(), &friendpb.DeleteFriendRequest{
		UserId:   userID,
		FriendId: friendID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateRemark 更新好友备注
// @Summary      更新好友备注
// @Description  更新指定好友的备注名称
// @Tags         好友
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "好友用户ID"
// @Param        request  body  object  true  "备注信息"
// @Success      200  {object}  response.Response  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /friends/{id}/remark [put]
func (h *FriendHandler) UpdateRemark(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	friendID := c.Param("id")

	var req struct {
		Remark string `json:"remark" binding:"max=50" example:"老朋友"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err := h.clientManager.Friend().UpdateRemark(c.Request.Context(), &friendpb.UpdateRemarkRequest{
		UserId:   userID,
		FriendId: friendID,
		Remark:   req.Remark,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// AddToBlacklist 添加黑名单
// @Summary      添加黑名单
// @Description  将指定用户添加到黑名单
// @Tags         好友
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  object  true  "用户ID"
// @Success      200  {object}  response.Response  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /friends/blacklist [post]
func (h *FriendHandler) AddToBlacklist(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req struct {
		UserID string `json:"userId" binding:"required" example:"user-456"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	_, err := h.clientManager.Friend().AddToBlacklist(c.Request.Context(), &friendpb.AddToBlacklistRequest{
		UserId:        userID,
		BlockedUserId: req.UserID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// RemoveFromBlacklist 从黑名单移除
// @Summary      从黑名单移除
// @Description  将指定用户从黑名单移除
// @Tags         好友
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "被拉黑用户ID"
// @Success      200  {object}  response.Response  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /friends/blacklist/{id} [delete]
func (h *FriendHandler) RemoveFromBlacklist(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	blockedUserID := c.Param("id")

	_, err := h.clientManager.Friend().RemoveFromBlacklist(c.Request.Context(), &friendpb.RemoveFromBlacklistRequest{
		UserId:        userID,
		BlockedUserId: blockedUserID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// GetBlacklist 获取黑名单
// @Summary      获取黑名单
// @Description  获取当前用户的黑名单列表
// @Tags         好友
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /friends/blacklist [get]
func (h *FriendHandler) GetBlacklist(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	resp, err := h.clientManager.Friend().GetBlacklist(c.Request.Context(), &friendpb.GetBlacklistRequest{
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}
