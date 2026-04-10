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

const (
	defaultMessageAnchorLimit = 20
	maxMessageAnchorLimit     = 100
)

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

func parseMessageAnchorLimit(c *gin.Context, key string, defaultValue int32) (int32, bool) {
	raw := c.Query(key)
	if raw == "" {
		return defaultValue, true
	}

	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		response.ParamError(c, key+" must be an integer")
		return 0, false
	}
	if value <= 0 {
		return defaultValue, true
	}
	if value > maxMessageAnchorLimit {
		value = maxMessageAnchorLimit
	}

	return int32(value), true
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

// GetMessagesBefore gets messages before anchor message
// @Summary      get messages before anchor
// @Description  Query messages before an anchor_message_id in a conversation
// @Tags         message
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId      path      string  true   "conversation ID"
// @Param        anchor_message_id   query     string  true   "anchor message ID"
// @Param        limit               query     int32   false  "limit (default 20, max 100)"
// @Success      200                 {object}  response.Response{data=object}  "success"
// @Failure      400                 {object}  response.Response  "parameter error"
// @Failure      401                 {object}  response.Response  "unauthorized"
// @Failure      500                 {object}  response.Response  "server error"
// @Router       /conversations/{conversationId}/messages/before [get]
func (h *MessageHandler) GetMessagesBefore(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		response.ParamError(c, "conversation_id is required")
		return
	}
	if !h.ensureConversationAccessible(c, userID, conversationID) {
		return
	}

	anchorMessageID := c.Query("anchor_message_id")
	if anchorMessageID == "" {
		response.ParamError(c, "anchor_message_id is required")
		return
	}

	limit, ok := parseMessageAnchorLimit(c, "limit", defaultMessageAnchorLimit)
	if !ok {
		return
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	resp, err := h.clientManager.Message().GetMessagesBefore(ctx, &messagepb.GetMessagesBeforeRequest{
		ConversationId:  conversationID,
		AnchorMessageId: anchorMessageID,
		Limit:           limit,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// GetMessagesAfter gets messages after anchor message
// @Summary      get messages after anchor
// @Description  Query messages after an anchor_message_id in a conversation
// @Tags         message
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId      path      string  true   "conversation ID"
// @Param        anchor_message_id   query     string  true   "anchor message ID"
// @Param        limit               query     int32   false  "limit (default 20, max 100)"
// @Success      200                 {object}  response.Response{data=object}  "success"
// @Failure      400                 {object}  response.Response  "parameter error"
// @Failure      401                 {object}  response.Response  "unauthorized"
// @Failure      500                 {object}  response.Response  "server error"
// @Router       /conversations/{conversationId}/messages/after [get]
func (h *MessageHandler) GetMessagesAfter(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		response.ParamError(c, "conversation_id is required")
		return
	}
	if !h.ensureConversationAccessible(c, userID, conversationID) {
		return
	}

	anchorMessageID := c.Query("anchor_message_id")
	if anchorMessageID == "" {
		response.ParamError(c, "anchor_message_id is required")
		return
	}

	limit, ok := parseMessageAnchorLimit(c, "limit", defaultMessageAnchorLimit)
	if !ok {
		return
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	resp, err := h.clientManager.Message().GetMessagesAfter(ctx, &messagepb.GetMessagesAfterRequest{
		ConversationId:  conversationID,
		AnchorMessageId: anchorMessageID,
		Limit:           limit,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// GetMessagesAroundAnchor gets messages around anchor message
// @Summary      get messages around anchor
// @Description  Query messages before and after anchor_message_id in one request
// @Tags         message
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId      path      string  true   "conversation ID"
// @Param        anchor_message_id   query     string  true   "anchor message ID"
// @Param        before              query     int32   false  "messages before anchor (default 20, max 100)"
// @Param        after               query     int32   false  "messages after anchor (default 20, max 100)"
// @Param        include_anchor      query     bool    false  "include anchor message (default true)"
// @Success      200                 {object}  response.Response{data=object}  "success"
// @Failure      400                 {object}  response.Response  "parameter error"
// @Failure      401                 {object}  response.Response  "unauthorized"
// @Failure      500                 {object}  response.Response  "server error"
// @Router       /conversations/{conversationId}/messages/around-anchor [get]
func (h *MessageHandler) GetMessagesAroundAnchor(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		response.ParamError(c, "conversation_id is required")
		return
	}
	if !h.ensureConversationAccessible(c, userID, conversationID) {
		return
	}

	anchorMessageID := c.Query("anchor_message_id")
	if anchorMessageID == "" {
		response.ParamError(c, "anchor_message_id is required")
		return
	}

	beforeLimit, ok := parseMessageAnchorLimit(c, "before", defaultMessageAnchorLimit)
	if !ok {
		return
	}
	afterLimit, ok := parseMessageAnchorLimit(c, "after", defaultMessageAnchorLimit)
	if !ok {
		return
	}

	includeAnchor := true
	if includeAnchorStr := c.Query("include_anchor"); includeAnchorStr != "" {
		parsed, err := strconv.ParseBool(includeAnchorStr)
		if err != nil {
			response.ParamError(c, "include_anchor must be a boolean")
			return
		}
		includeAnchor = parsed
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	resp, err := h.clientManager.Message().GetMessagesAroundAnchor(ctx, &messagepb.GetMessagesAroundAnchorRequest{
		ConversationId:  conversationID,
		AnchorMessageId: anchorMessageID,
		Before:          beforeLimit,
		After:           afterLimit,
		IncludeAnchor:   &includeAnchor,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// GetFirstUnreadAnchor gets first unread anchor message
// @Summary      get first unread anchor
// @Description  Query first unread message anchor, optionally returning context windows
// @Tags         message
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversationId  path      string  true   "conversation ID"
// @Param        with_context    query     bool    false  "whether to include before/after context"
// @Param        before          query     int32   false  "messages before anchor when with_context=true (default 20, max 100)"
// @Param        after           query     int32   false  "messages after anchor when with_context=true (default 20, max 100)"
// @Success      200             {object}  response.Response{data=object}  "success"
// @Failure      400             {object}  response.Response  "parameter error"
// @Failure      401             {object}  response.Response  "unauthorized"
// @Failure      500             {object}  response.Response  "server error"
// @Router       /conversations/{conversationId}/messages/first-unread-anchor [get]
func (h *MessageHandler) GetFirstUnreadAnchor(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		response.ParamError(c, "conversation_id is required")
		return
	}
	if !h.ensureConversationAccessible(c, userID, conversationID) {
		return
	}

	var withContext *bool
	if withContextStr := c.Query("with_context"); withContextStr != "" {
		parsed, err := strconv.ParseBool(withContextStr)
		if err != nil {
			response.ParamError(c, "with_context must be a boolean")
			return
		}
		withContext = &parsed
	}

	req := &messagepb.GetFirstUnreadAnchorRequest{
		ConversationId: conversationID,
		WithContext:    withContext,
	}

	if withContext != nil && *withContext {
		beforeLimit, ok := parseMessageAnchorLimit(c, "before", defaultMessageAnchorLimit)
		if !ok {
			return
		}
		afterLimit, ok := parseMessageAnchorLimit(c, "after", defaultMessageAnchorLimit)
		if !ok {
			return
		}
		req.Before = &beforeLimit
		req.After = &afterLimit
	}

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	resp, err := h.clientManager.Message().GetFirstUnreadAnchor(ctx, req)
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

	ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-user-id", userID)
	resp, err := h.clientManager.Message().GetMessageById(ctx, &messagepb.GetMessageByIdRequest{
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
