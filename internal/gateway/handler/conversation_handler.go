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

// ConversationHandler conversation HTTP handler
type ConversationHandler struct {
	clientManager *client.Manager
}

// NewConversationHandler creates conversation handler
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

// GetConversations get conversation list
// @Summary      get conversation list
// @Description  Get current user's conversation list, supports incremental sync (via updatedBefore parameter)
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit          query  int    false  "return count (default 20, max 100)"
// @Param        updatedBefore  query  int64  false  "Unix timestamp, only return conversations updated before this time (incremental sync)"
// @Success      200  {object}  response.Response{data=object}  "success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// GetConversation get single conversation
// @Summary      get single conversation
// @Description  Get details of specified conversation
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string  true  "conversation ID"
// @Success      200  {object}  response.Response{data=object}  "success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      404  {object}  response.Response  "conversation not found"
// @Failure      500  {object}  response.Response  "server error"
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

// DeleteConversation delete conversation
// @Summary      delete conversation
// @Description  Delete specified conversation (doesn't affect messages, only removes from conversation list)
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string  true  "conversation ID"
// @Success      200  {object}  response.Response  "success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// setPinnedRequest pin request body
type setPinnedRequest struct {
	Pinned bool `json:"pinned"`
}

// SetPinned set conversation pin
// @Summary      set conversation pin
// @Description  Pin or unpin specified conversation
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string           true  "conversation ID"
// @Param        request    body  setPinnedRequest  true  "pin status"
// @Success      200  {object}  response.Response  "success"
// @Failure      400  {object}  response.Response  "parameter error"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// setMutedRequest mute request body
type setMutedRequest struct {
	Muted bool `json:"muted"`
}

// SetMuted set conversation mute
// @Summary      set conversation mute
// @Description  Enable or disable mute for specified conversation
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string          true  "conversation ID"
// @Param        request    body  setMutedRequest  true  "mute status"
// @Success      200  {object}  response.Response  "success"
// @Failure      400  {object}  response.Response  "parameter error"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// MarkRead mark conversation as read (clear unread count)
// @Summary      mark conversation as read
// @Description  Clear unread count for specified conversation and advance message read cursor to latest sequence
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string  true  "conversation ID"
// @Success      200  {object}  response.Response  "success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// MarkMessagesRead batch mark read by message ID
// @Summary      mark read by message ID
// @Description  Used in scrolling list scenarios, batch report visible message IDs and advance conversation read cursor
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string                   true  "conversation ID"
// @Param        request         body  markMessagesReadRequest  true  "message read list"
// @Success      200  {object}  response.Response{data=object}  "success"
// @Failure      400  {object}  response.Response  "parameter error"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// GetMessageUnreadCount get conversation unread count
// @Summary      get conversation unread count
// @Description  Query unread message count for current user in specified conversation
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path      string  true   "conversation ID"
// @Param        last_read_seq   query     int64   false  "optional, client read sequence"
// @Success      200             {object}  response.Response{data=object}  "success"
// @Failure      400             {object}  response.Response  "parameter error"
// @Failure      401             {object}  response.Response  "unauthorized"
// @Failure      500             {object}  response.Response  "server error"
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

// GetMessageReadReceipts get conversation message read receipts
// @Summary      get message read receipts
// @Description  Return last read sequence for members in conversation
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path      string  true  "conversation ID"
// @Success      200             {object}  response.Response{data=object}  "success"
// @Failure      400             {object}  response.Response  "parameter error"
// @Failure      401             {object}  response.Response  "unauthorized"
// @Failure      500             {object}  response.Response  "server error"
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

// GetMessageSequence get conversation message sequence
// @Summary      get conversation message sequence
// @Description  Get latest message sequence in conversation
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path      string  true  "conversation ID"
// @Success      200             {object}  response.Response{data=object}  "success"
// @Failure      400             {object}  response.Response  "parameter error"
// @Failure      401             {object}  response.Response  "unauthorized"
// @Failure      500             {object}  response.Response  "server error"
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

// GetTotalUnread get total unread count
// @Summary      get total unread count
// @Description  Get total unread message count for all conversations of current user (muted conversations excluded)
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=object}  "success"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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

// setBurnAfterReadingRequest burn after reading request body
type setBurnAfterReadingRequest struct {
	Duration int32 `json:"duration"` // seconds, 0 means cancel
}

// SetBurnAfterReading set burn after reading
// @Summary      set burn after reading
// @Description  Set conversation burn after reading duration, 0 means cancel
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string                     true  "conversation ID"
// @Param        request    body  setBurnAfterReadingRequest  true  "burn after reading duration (seconds)"
// @Success      200  {object}  response.Response  "success"
// @Failure      400  {object}  response.Response  "parameter error"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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
	Duration int32 `json:"duration"` // seconds, 0 means cancel
}

// SetAutoDelete set auto delete
// @Summary      set auto delete
// @Description  Set conversation auto delete duration, 0 means cancel
// @Tags         conversation
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path  string                  true  "conversation ID"
// @Param        request    body  setAutoDeleteRequest   true  "auto delete duration (seconds)"
// @Success      200  {object}  response.Response  "success"
// @Failure      400  {object}  response.Response  "parameter error"
// @Failure      401  {object}  response.Response  "unauthorized"
// @Failure      500  {object}  response.Response  "server error"
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
