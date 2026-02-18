package handler

import (
	"net/http"
	"strconv"

	sessionpb "github.com/anychat/server/api/proto/session"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// SessionHandler session HTTP处理器
type SessionHandler struct {
	clientManager *client.Manager
}

// NewSessionHandler 创建session处理器
func NewSessionHandler(clientManager *client.Manager) *SessionHandler {
	return &SessionHandler{clientManager: clientManager}
}

// GetSessions 获取会话列表
// @Summary      获取会话列表
// @Description  获取当前用户的会话列表，支持增量同步（通过updatedBefore参数）
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit          query  int    false  "返回数量（默认20，最大100）"
// @Param        updatedBefore  query  int64  false  "Unix时间戳，仅返回此时间之前更新的会话（增量同步）"
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /sessions [get]
func (h *SessionHandler) GetSessions(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	req := &sessionpb.GetSessionsRequest{UserId: userID}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = int32(limit)
		}
	}
	if beforeStr := c.Query("updatedBefore"); beforeStr != "" {
		if t, err := strconv.ParseInt(beforeStr, 10, 64); err == nil {
			req.UpdatedBefore = &t
		}
	}

	resp, err := h.clientManager.Session().GetSessions(c.Request.Context(), req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// GetSession 获取单个会话
// @Summary      获取单个会话
// @Description  获取指定会话的详情
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId  path  string  true  "会话ID"
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      404  {object}  response.Response  "会话不存在"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /sessions/{sessionId} [get]
func (h *SessionHandler) GetSession(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	sessionID := c.Param("sessionId")

	resp, err := h.clientManager.Session().GetSession(c.Request.Context(), &sessionpb.GetSessionRequest{
		UserId:    userID,
		SessionId: sessionID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// DeleteSession 删除会话
// @Summary      删除会话
// @Description  删除指定会话（不影响消息，仅从会话列表中移除）
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId  path  string  true  "会话ID"
// @Success      200  {object}  response.Response  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /sessions/{sessionId} [delete]
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	sessionID := c.Param("sessionId")

	_, err := h.clientManager.Session().DeleteSession(c.Request.Context(), &sessionpb.DeleteSessionRequest{
		UserId:    userID,
		SessionId: sessionID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// setPinnedRequest 置顶请求体
type setPinnedRequest struct {
	Pinned bool `json:"pinned"`
}

// SetPinned 设置会话置顶
// @Summary      设置会话置顶
// @Description  置顶或取消置顶指定会话
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId  path  string           true  "会话ID"
// @Param        request    body  setPinnedRequest  true  "置顶状态"
// @Success      200  {object}  response.Response  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /sessions/{sessionId}/pin [put]
func (h *SessionHandler) SetPinned(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	sessionID := c.Param("sessionId")

	var req setPinnedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.clientManager.Session().SetPinned(c.Request.Context(), &sessionpb.SetPinnedRequest{
		UserId:    userID,
		SessionId: sessionID,
		Pinned:    req.Pinned,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// setMutedRequest 免打扰请求体
type setMutedRequest struct {
	Muted bool `json:"muted"`
}

// SetMuted 设置会话免打扰
// @Summary      设置会话免打扰
// @Description  开启或关闭指定会话的免打扰
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId  path  string          true  "会话ID"
// @Param        request    body  setMutedRequest  true  "免打扰状态"
// @Success      200  {object}  response.Response  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /sessions/{sessionId}/mute [put]
func (h *SessionHandler) SetMuted(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	sessionID := c.Param("sessionId")

	var req setMutedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.clientManager.Session().SetMuted(c.Request.Context(), &sessionpb.SetMutedRequest{
		UserId:    userID,
		SessionId: sessionID,
		Muted:     req.Muted,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// MarkRead 标记会话已读（清除未读数）
// @Summary      标记会话已读
// @Description  清除指定会话的未读消息数
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        sessionId  path  string  true  "会话ID"
// @Success      200  {object}  response.Response  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /sessions/{sessionId}/read [post]
func (h *SessionHandler) MarkRead(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	sessionID := c.Param("sessionId")

	_, err := h.clientManager.Session().ClearUnread(c.Request.Context(), &sessionpb.ClearUnreadRequest{
		UserId:    userID,
		SessionId: sessionID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetTotalUnread 获取总未读数
// @Summary      获取总未读数
// @Description  获取当前用户所有会话的总未读消息数（免打扰会话不计入）
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /sessions/unread/total [get]
func (h *SessionHandler) GetTotalUnread(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	resp, err := h.clientManager.Session().GetTotalUnread(c.Request.Context(), &sessionpb.GetTotalUnreadRequest{
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}
