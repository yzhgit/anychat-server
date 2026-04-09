package handler

import (
	"strconv"

	conversationpb "github.com/anychat/server/api/proto/conversation"
	messagepb "github.com/anychat/server/api/proto/message"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
)

// MessageHandler message HTTP handler
type MessageHandler struct {
	clientManager *client.Manager
}

// NewMessageHandler creates message handler
func NewMessageHandler(clientManager *client.Manager) *MessageHandler {
	return &MessageHandler{
		clientManager: clientManager,
	}
}

func (h *MessageHandler) ensureConversationAccessible(c *gin.Context, userID, conversationID string) bool {
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

type sendMessageRequest struct {
	ConversationID string   `json:"conversation_id" binding:"required"`
	ContentType    string   `json:"content_type" binding:"required"`
	Content        string   `json:"content" binding:"required"`
	ReplyTo        *string  `json:"reply_to,omitempty"`
	AtUsers        []string `json:"at_users,omitempty"`
	LocalID        string   `json:"local_id" binding:"required"`
}

type recallMessageRequest struct {
	MessageID string `json:"message_id" binding:"required"`
}

type ackReadTriggersRequest struct {
	Events []readTriggerEvent `json:"events" binding:"required,min=1"`
}

type readTriggerEvent struct {
	MessageID      string `json:"message_id" binding:"required"`
	ClientAt       *int64 `json:"client_at,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// SendMessage send message
// @Summary      send message
// @Description  Send conversation message via HTTP (supports idempotent local_id)
// @Tags         message
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      sendMessageRequest  true  "message content"
// @Success      200      {object}  response.Response{data=object}  "success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /messages [post]
func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	grpcReq := &messagepb.SendMessageRequest{
		SenderId:       userID,
		ConversationId: req.ConversationID,
		ContentType:    req.ContentType,
		Content:        req.Content,
		LocalId:        req.LocalID,
		AtUsers:        req.AtUsers,
	}
	if req.ReplyTo != nil && *req.ReplyTo != "" {
		grpcReq.ReplyTo = req.ReplyTo
	}

	resp, err := h.clientManager.Message().SendMessage(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// GetMessages get message list
// @Summary      get message history
// @Description  Paginate pull messages by conversation ID and sequence range
// @Tags         message
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversation_id  query     string  true   "conversation ID"
// @Param        start_seq        query     int64   false  "start sequence"
// @Param        end_seq          query     int64   false  "end sequence"
// @Param        limit            query     int32   false  "limit (default 20, max 100)"
// @Param        reverse          query     bool    false  "reverse order"
// @Success      200              {object}  response.Response{data=object}  "success"
// @Failure      400              {object}  response.Response  "parameter error"
// @Failure      401              {object}  response.Response  "unauthorized"
// @Failure      500              {object}  response.Response  "server error"
// @Router       /messages [get]
func (h *MessageHandler) GetMessages(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Query("conversation_id")
	if conversationID == "" {
		response.ParamError(c, "conversation_id is required")
		return
	}
	if !h.ensureConversationAccessible(c, userID, conversationID) {
		return
	}

	req := &messagepb.GetMessagesRequest{
		ConversationId: conversationID,
	}

	if startSeqStr := c.Query("start_seq"); startSeqStr != "" {
		startSeq, err := strconv.ParseInt(startSeqStr, 10, 64)
		if err != nil {
			response.ParamError(c, "start_seq must be an integer")
			return
		}
		req.StartSeq = &startSeq
	}

	if endSeqStr := c.Query("end_seq"); endSeqStr != "" {
		endSeq, err := strconv.ParseInt(endSeqStr, 10, 64)
		if err != nil {
			response.ParamError(c, "end_seq must be an integer")
			return
		}
		req.EndSeq = &endSeq
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil {
			response.ParamError(c, "limit must be an integer")
			return
		}
		req.Limit = int32(limit)
	}

	if reverseStr := c.Query("reverse"); reverseStr != "" {
		reverse, err := strconv.ParseBool(reverseStr)
		if err != nil {
			response.ParamError(c, "reverse must be a boolean")
			return
		}
		req.Reverse = reverse
	}

	resp, err := h.clientManager.Message().GetMessages(c.Request.Context(), req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// GetMessageByID get single message
// @Summary      get message detail
// @Description  Get single message by message ID
// @Tags         message
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        messageId  path      string  true  "message ID"
// @Success      200        {object}  response.Response{data=object}  "success"
// @Failure      400        {object}  response.Response  "parameter error"
// @Failure      401        {object}  response.Response  "unauthorized"
// @Failure      404        {object}  response.Response  "message not found"
// @Failure      500        {object}  response.Response  "server error"
// @Router       /messages/{messageId} [get]
func (h *MessageHandler) GetMessageByID(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	messageID := c.Param("messageId")
	if messageID == "" {
		response.ParamError(c, "message_id is required")
		return
	}

	resp, err := h.clientManager.Message().GetMessageById(c.Request.Context(), &messagepb.GetMessageByIdRequest{
		MessageId: messageID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}
	if !h.ensureConversationAccessible(c, userID, resp.GetConversationId()) {
		return
	}

	response.Success(c, resp)
}

// SearchMessages search conversation messages
// @Summary      search messages
// @Description  Search messages by keyword and conversation scope
// @Tags         message
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        keyword          query     string  true   "keyword"
// @Param        conversation_id  query     string  true   "conversation ID"
// @Param        content_type     query     string  false  "message type"
// @Param        limit            query     int32   false  "page size (default 20, max 100)"
// @Param        offset           query     int32   false  "offset"
// @Success      200              {object}  response.Response{data=object}  "success"
// @Failure      400              {object}  response.Response  "parameter error"
// @Failure      401              {object}  response.Response  "unauthorized"
// @Failure      500              {object}  response.Response  "server error"
// @Router       /messages/search [get]
func (h *MessageHandler) SearchMessages(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	keyword := c.Query("keyword")
	if keyword == "" {
		response.ParamError(c, "keyword is required")
		return
	}

	conversationID := c.Query("conversation_id")
	if conversationID == "" {
		response.ParamError(c, "conversation_id is required")
		return
	}

	req := &messagepb.SearchMessagesRequest{
		Keyword:        keyword,
		ConversationId: &conversationID,
	}

	if contentType := c.Query("content_type"); contentType != "" {
		req.ContentType = &contentType
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil {
			response.ParamError(c, "limit must be an integer")
			return
		}
		req.Limit = int32(limit)
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.ParseInt(offsetStr, 10, 32)
		if err != nil {
			response.ParamError(c, "offset must be an integer")
			return
		}
		req.Offset = int32(offset)
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	resp, err := h.clientManager.Message().SearchMessages(ctx, req)
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// RecallMessage recall message
// @Summary      recall message
// @Description  Recall specified message, can only recall own messages within 2 minutes
// @Tags         message
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      recallMessageRequest  true  "message ID"
// @Success      200      {object}  response.Response  "success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      403      {object}  response.Response  "no permission or timeout"
// @Failure      404      {object}  response.Response  "message not found"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /messages/recall [post]
func (h *MessageHandler) RecallMessage(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req recallMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	_, err := h.clientManager.Message().RecallMessage(ctx, &messagepb.RecallMessageRequest{
		MessageId: req.MessageID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// DeleteMessage delete message
// @Summary      delete message
// @Description  Delete specified message, can only delete own messages
// @Tags         message
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        messageId  path      string  true  "message ID"
// @Success      200      {object}  response.Response  "success"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      403      {object}  response.Response  "no permission"
// @Failure      404      {object}  response.Response  "message not found"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /messages/{messageId} [delete]
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	messageID := c.Param("messageId")

	if messageID == "" {
		response.ParamError(c, "message_id is required")
		return
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	_, err := h.clientManager.Message().DeleteMessage(ctx, &messagepb.DeleteMessageRequest{
		MessageId: messageID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, nil)
}

// AckReadTriggers burn after reading read trigger acknowledgment
// @Summary      burn after reading read trigger acknowledgment
// @Description  Client batch reports message read trigger events, server starts burn timer accordingly
// @Tags         message
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      ackReadTriggersRequest  true  "read trigger events"
// @Success      200      {object}  response.Response{data=object}  "success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /messages/read-triggers [post]
func (h *MessageHandler) AckReadTriggers(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req ackReadTriggersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	events := make([]*messagepb.ReadTriggerEvent, 0, len(req.Events))
	for _, event := range req.Events {
		pbEvent := &messagepb.ReadTriggerEvent{
			MessageId: event.MessageID,
		}
		if event.ClientAt != nil {
			pbEvent.ClientAt = event.ClientAt
		}
		if event.IdempotencyKey != "" {
			pbEvent.IdempotencyKey = &event.IdempotencyKey
		}
		events = append(events, pbEvent)
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	resp, err := h.clientManager.Message().AckReadTriggers(ctx, &messagepb.AckReadTriggersRequest{
		Events: events,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"success_ids": resp.SuccessIds,
		"ignored_ids": resp.IgnoredIds,
	})
}
