package handler

import (
	"net/http"
	"strconv"

	conversationpb "github.com/anychat/server/api/proto/conversation"
	messagepb "github.com/anychat/server/api/proto/message"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
)

// ConversationHandler conversation HTTP处理器
type ConversationHandler struct {
	clientManager *client.Manager
}

// NewConversationHandler 创建conversation处理器
func NewConversationHandler(clientManager *client.Manager) *ConversationHandler {
	return &ConversationHandler{clientManager: clientManager}
}

func (h *ConversationHandler) ensureConversationAccessible(c *gin.Context, userID, conversationID string) bool {
	_, err := h.clientManager.Conversation().GetConversation(c.Request.Context(), &conversationpb.GetConversationRequest{
		UserId:         userID,
		ConversationId: conversationID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return false
	}
	return true
}

// GetConversations 获取会话列表
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
// @Router       /conversations [get]
func (h *ConversationHandler) GetConversations(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	req := &conversationpb.GetConversationsRequest{UserId: userID}

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

	resp, err := h.clientManager.Conversation().GetConversations(c.Request.Context(), req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// GetConversation 获取单个会话
// @Summary      获取单个会话
// @Description  获取指定会话的详情
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string  true  "会话ID"
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      404  {object}  response.Response  "会话不存在"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /conversations/{conversationId} [get]
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")

	resp, err := h.clientManager.Conversation().GetConversation(c.Request.Context(), &conversationpb.GetConversationRequest{
		UserId:         userID,
		ConversationId: conversationID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// DeleteConversation 删除会话
// @Summary      删除会话
// @Description  删除指定会话（不影响消息，仅从会话列表中移除）
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string  true  "会话ID"
// @Success      200  {object}  response.Response  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /conversations/{conversationId} [delete]
func (h *ConversationHandler) DeleteConversation(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")

	_, err := h.clientManager.Conversation().DeleteConversation(c.Request.Context(), &conversationpb.DeleteConversationRequest{
		UserId:         userID,
		ConversationId: conversationID,
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
// @Param        conversationId  path  string           true  "会话ID"
// @Param        request    body  setPinnedRequest  true  "置顶状态"
// @Success      200  {object}  response.Response  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /conversations/{conversationId}/pin [put]
func (h *ConversationHandler) SetPinned(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")

	var req setPinnedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.clientManager.Conversation().SetPinned(c.Request.Context(), &conversationpb.SetPinnedRequest{
		UserId:         userID,
		ConversationId: conversationID,
		Pinned:         req.Pinned,
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
// @Param        conversationId  path  string          true  "会话ID"
// @Param        request    body  setMutedRequest  true  "免打扰状态"
// @Success      200  {object}  response.Response  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /conversations/{conversationId}/mute [put]
func (h *ConversationHandler) SetMuted(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")

	var req setMutedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.clientManager.Conversation().SetMuted(c.Request.Context(), &conversationpb.SetMutedRequest{
		UserId:         userID,
		ConversationId: conversationID,
		Muted:          req.Muted,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

// MarkRead 标记会话已读（清除未读数）
// @Summary      标记会话已读
// @Description  清除指定会话未读数，并同步推进消息已读游标到当前最新序列
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string  true  "会话ID"
// @Success      200  {object}  response.Response  "成功"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /conversations/{conversationId}/read-all [post]
func (h *ConversationHandler) MarkRead(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")

	if !h.ensureConversationAccessible(c, userID, conversationID) {
		return
	}

	seqResp, err := h.clientManager.Message().GetConversationSequence(c.Request.Context(), &messagepb.GetConversationSequenceRequest{
		ConversationId: conversationID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	_, err = h.clientManager.Message().MarkAsRead(ctx, &messagepb.MarkAsReadRequest{
		ConversationId: conversationID,
		LastReadSeq:    seqResp.CurrentSeq,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	_, err = h.clientManager.Conversation().ClearUnread(c.Request.Context(), &conversationpb.ClearUnreadRequest{
		UserId:         userID,
		ConversationId: conversationID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

type markMessagesReadRequest struct {
	MessageIDs     []string `json:"message_ids" binding:"required,min=1"`
	ClientReadAt   *int64   `json:"client_read_at,omitempty"`
	IdempotencyKey *string  `json:"idempotency_key,omitempty"`
}

// MarkMessagesRead 批量按消息ID标记已读
// @Summary      按消息ID标记已读
// @Description  用于滚动列表等场景，批量上报可见消息ID并推进会话已读游标
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string                   true  "会话ID"
// @Param        request         body  markMessagesReadRequest  true  "消息已读列表"
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /conversations/{conversationId}/messages/read [post]
func (h *ConversationHandler) MarkMessagesRead(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		response.ParamError(c, "conversation_id is required")
		return
	}

	var req markMessagesReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	grpcReq := &messagepb.MarkMessagesReadRequest{
		ConversationId: conversationID,
		MessageIds:     req.MessageIDs,
	}
	if req.ClientReadAt != nil {
		grpcReq.ClientReadAt = req.ClientReadAt
	}
	if req.IdempotencyKey != nil && *req.IdempotencyKey != "" {
		grpcReq.IdempotencyKey = req.IdempotencyKey
	}

	resp, err := h.clientManager.Message().MarkMessagesRead(ctx, grpcReq)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// GetMessageUnreadCount 获取会话未读数
// @Summary      获取会话未读数
// @Description  查询当前用户在指定会话的未读消息数
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path      string  true   "会话ID"
// @Param        last_read_seq   query     int64   false  "可选，客户端已读序列号"
// @Success      200             {object}  response.Response{data=object}  "成功"
// @Failure      400             {object}  response.Response  "参数错误"
// @Failure      401             {object}  response.Response  "未授权"
// @Failure      500             {object}  response.Response  "服务器错误"
// @Router       /conversations/{conversationId}/messages/unread-count [get]
func (h *ConversationHandler) GetMessageUnreadCount(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		response.ParamError(c, "conversation_id is required")
		return
	}

	req := &messagepb.GetUnreadCountRequest{
		ConversationId: conversationID,
	}
	if lastReadSeqStr := c.Query("last_read_seq"); lastReadSeqStr != "" {
		lastReadSeq, err := strconv.ParseInt(lastReadSeqStr, 10, 64)
		if err != nil {
			response.ParamError(c, "last_read_seq must be an integer")
			return
		}
		req.LastReadSeq = &lastReadSeq
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	resp, err := h.clientManager.Message().GetUnreadCount(ctx, req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// GetMessageReadReceipts 获取会话消息已读回执
// @Summary      获取消息已读回执
// @Description  返回会话中成员的最后已读序列
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path      string  true  "会话ID"
// @Success      200             {object}  response.Response{data=object}  "成功"
// @Failure      400             {object}  response.Response  "参数错误"
// @Failure      401             {object}  response.Response  "未授权"
// @Failure      500             {object}  response.Response  "服务器错误"
// @Router       /conversations/{conversationId}/messages/read-receipts [get]
func (h *ConversationHandler) GetMessageReadReceipts(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		response.ParamError(c, "conversation_id is required")
		return
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	resp, err := h.clientManager.Message().GetReadReceipts(ctx, &messagepb.GetReadReceiptsRequest{
		ConversationId: conversationID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// GetMessageSequence 获取会话消息序列号
// @Summary      获取会话消息序列号
// @Description  获取会话中当前最新消息序列号
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path      string  true  "会话ID"
// @Success      200             {object}  response.Response{data=object}  "成功"
// @Failure      400             {object}  response.Response  "参数错误"
// @Failure      401             {object}  response.Response  "未授权"
// @Failure      500             {object}  response.Response  "服务器错误"
// @Router       /conversations/{conversationId}/messages/sequence [get]
func (h *ConversationHandler) GetMessageSequence(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		response.ParamError(c, "conversation_id is required")
		return
	}
	if !h.ensureConversationAccessible(c, userID, conversationID) {
		return
	}

	resp, err := h.clientManager.Message().GetConversationSequence(c.Request.Context(), &messagepb.GetConversationSequenceRequest{
		ConversationId: conversationID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
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
// @Router       /conversations/unread/total [get]
func (h *ConversationHandler) GetTotalUnread(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	resp, err := h.clientManager.Conversation().GetTotalUnread(c.Request.Context(), &conversationpb.GetTotalUnreadRequest{
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, resp)
}

// setBurnAfterReadingRequest 阅后即焚请求体
type setBurnAfterReadingRequest struct {
	Duration int32 `json:"duration"` // 秒,0表示取消
}

// SetBurnAfterReading 设置阅后即焚
// @Summary      设置阅后即焚
// @Description  设置会话阅后即焚时长，0表示取消
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string                     true  "会话ID"
// @Param        request    body  setBurnAfterReadingRequest  true  "阅后即焚时长(秒)"
// @Success      200  {object}  response.Response  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /conversations/{conversationId}/burn [put]
func (h *ConversationHandler) SetBurnAfterReading(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")

	var req setBurnAfterReadingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.clientManager.Conversation().SetBurnAfterReading(c.Request.Context(), &conversationpb.SetBurnAfterReadingRequest{
		UserId:         userID,
		ConversationId: conversationID,
		Duration:       req.Duration,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}

type setAutoDeleteRequest struct {
	Duration int32 `json:"duration"` // 秒,0表示取消
}

// SetAutoDelete 设置自动删除
// @Summary      设置自动删除
// @Description  设置会话自动删除时长，0表示取消
// @Tags         会话
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string                  true  "会话ID"
// @Param        request    body  setAutoDeleteRequest   true  "自动删除时长(秒)"
// @Success      200  {object}  response.Response  "成功"
// @Failure      400  {object}  response.Response  "参数错误"
// @Failure      401  {object}  response.Response  "未授权"
// @Failure      500  {object}  response.Response  "服务器错误"
// @Router       /conversations/{conversationId}/auto_delete [put]
func (h *ConversationHandler) SetAutoDelete(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")

	var req setAutoDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.clientManager.Conversation().SetAutoDelete(c.Request.Context(), &conversationpb.SetAutoDeleteRequest{
		UserId:         userID,
		ConversationId: conversationID,
		Duration:       req.Duration,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	response.Success(c, nil)
}
