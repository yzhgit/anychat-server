package handler

import (
	"strconv"

	friendpb "github.com/anychat/server/api/proto/friend"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// FriendHandler friend HTTP handler
type FriendHandler struct {
	clientManager *client.Manager
}

// NewFriendHandler creates friend handler
func NewFriendHandler(clientManager *client.Manager) *FriendHandler {
	return &FriendHandler{
		clientManager: clientManager,
	}
}

// GetFriends get friend list
// @Summary      get friend list
// @Description  Get all friends of current user, supports incremental sync
// @Tags         friend
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        lastUpdateTime  query  int64  false  "last update timestamp (incremental sync)"
// @Success      200  {object}  response.Response{data=object}  "success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
// @Router       /friends [get]
func (h *FriendHandler) GetFriends(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	// Parse query parameters
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

// SendFriendRequest send friend request
// @Summary      send friend request
// @Description  Send friend request to specified user
// @Tags         friend
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  object  true  "request info"
// @Success      200  {object}  response.Response{data=object}  "success"
// @Failure      400  {object}  response.Response  "parameter error"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// HandleFriendRequest handle friend request
// @Summary      handle friend request
// @Description  Accept or reject friend request
// @Tags         friend
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  int  true  "request ID"
// @Param        request  body  object  true  "handle action"
// @Success      200  {object}  response.Response  "success"
// @Failure      400  {object}  response.Response  "parameter error"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// GetFriendRequests get friend request list
// @Summary      get friend request list
// @Description  Get received or sent friend request list
// @Tags         friend
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        type  query  string  false  "type: received/sent"  default(received)
// @Success      200  {object}  response.Response{data=object}  "success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// DeleteFriend delete friend
// @Summary      delete friend
// @Description  Delete specified friend
// @Tags         friend
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "friend user ID"
// @Success      200  {object}  response.Response  "success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// UpdateRemark update friend remark
// @Summary      update friend remark
// @Description  Update remark name for specified friend
// @Tags         friend
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "friend user ID"
// @Param        request  body  object  true  "remark info"
// @Success      200  {object}  response.Response  "success"
// @Failure      400  {object}  response.Response  "parameter error"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// AddToBlacklist add to blacklist
// @Summary      add to blacklist
// @Description  Add specified user to blacklist
// @Tags         friend
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  object  true  "user ID"
// @Success      200  {object}  response.Response  "success"
// @Failure      400  {object}  response.Response  "parameter error"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// RemoveFromBlacklist remove from blacklist
// @Summary      remove from blacklist
// @Description  Remove specified user from blacklist
// @Tags         friend
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "blocked user ID"
// @Success      200  {object}  response.Response  "success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// GetBlacklist get blacklist
// @Summary      get blacklist
// @Description  Get current user's blacklist
// @Tags         friend
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=object}  "success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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
