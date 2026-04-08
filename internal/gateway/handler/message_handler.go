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

// MessageHandler 消息HTTP处理器
type MessageHandler struct {
	clientManager *client.Manager
}

// NewMessageHandler 创建消息处理器
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

// SendMessage 发送消息
// @Summary      发送消息
// @Description  通过HTTP发送会话消息（支持幂等local_id）
// @Tags         消息
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      sendMessageRequest  true  "消息内容"
// @Success      200      {object}  response.Response{data=object}  "成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
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

// GetMessages 获取消息列表
// @Summary      获取历史消息
// @Description  按会话ID和序列号区间分页拉取消息
// @Tags         消息
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        conversation_id  query     string  true   "会话ID"
// @Param        start_seq        query     int64   false  "起始序列号"
// @Param        end_seq          query     int64   false  "结束序列号"
// @Param        limit            query     int32   false  "数量限制(默认20,最大100)"
// @Param        reverse          query     bool    false  "是否倒序"
// @Success      200              {object}  response.Response{data=object}  "成功"
// @Failure      400              {object}  response.Response  "参数错误"
// @Failure      401              {object}  response.Response  "未授权"
// @Failure      500              {object}  response.Response  "服务器错误"
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

// GetMessageByID 获取单条消息
// @Summary      获取消息详情
// @Description  通过消息ID获取单条消息
// @Tags         消息
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        messageId  path      string  true  "消息ID"
// @Success      200        {object}  response.Response{data=object}  "成功"
// @Failure      400        {object}  response.Response  "参数错误"
// @Failure      401        {object}  response.Response  "未授权"
// @Failure      404        {object}  response.Response  "消息不存在"
// @Failure      500        {object}  response.Response  "服务器错误"
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

// SearchMessages 搜索会话消息
// @Summary      搜索消息
// @Description  按关键字和会话范围搜索消息
// @Tags         消息
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        keyword          query     string  true   "关键字"
// @Param        conversation_id  query     string  true   "会话ID"
// @Param        content_type     query     string  false  "消息类型"
// @Param        limit            query     int32   false  "每页数量(默认20,最大100)"
// @Param        offset           query     int32   false  "偏移量"
// @Success      200              {object}  response.Response{data=object}  "成功"
// @Failure      400              {object}  response.Response  "参数错误"
// @Failure      401              {object}  response.Response  "未授权"
// @Failure      500              {object}  response.Response  "服务器错误"
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

// RecallMessage 撤回消息
// @Summary      撤回消息
// @Description  撤回指定消息，只能撤回自己发送的消息，且需在2分钟内
// @Tags         消息
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      recallMessageRequest  true  "消息ID"
// @Success      200      {object}  response.Response  "成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      403      {object}  response.Response  "无权限或已超时"
// @Failure      404      {object}  response.Response  "消息不存在"
// @Failure      500      {object}  response.Response  "服务器错误"
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

// DeleteMessage 删除消息
// @Summary      删除消息
// @Description  删除指定消息，只能删除自己发送的消息
// @Tags         消息
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        messageId  path      string  true  "消息ID"
// @Success      200      {object}  response.Response  "成功"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      403      {object}  response.Response  "无权限"
// @Failure      404      {object}  response.Response  "消息不存在"
// @Failure      500      {object}  response.Response  "服务器错误"
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

// AckReadTriggers 阅后即焚阅读触发回执
// @Summary      阅后即焚阅读触发回执
// @Description  客户端批量上报消息阅读触发事件，服务端据此启动阅后即焚计时
// @Tags         消息
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      ackReadTriggersRequest  true  "阅读触发事件"
// @Success      200      {object}  response.Response{data=object}  "成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
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
