package grpc

import (
	"context"
	stderrors "errors"

	commonpb "github.com/anychat/server/api/proto/common"
	messagepb "github.com/anychat/server/api/proto/message"
	"github.com/anychat/server/internal/message/service"
	pkgerrors "github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const operatorUserIDMetadataKey = "x-user-id"

// Server Message gRPC服务器
type Server struct {
	messagepb.UnimplementedMessageServiceServer
	messageService service.MessageService
}

// NewServer 创建gRPC服务器
func NewServer(messageService service.MessageService) *Server {
	return &Server{
		messageService: messageService,
	}
}

// SendMessage 发送消息
func (s *Server) SendMessage(ctx context.Context, req *messagepb.SendMessageRequest) (*messagepb.SendMessageResponse, error) {
	logger.Info("SendMessage called",
		zap.String("senderId", req.SenderId),
		zap.String("conversationId", req.ConversationId),
		zap.String("contentType", req.ContentType))

	// 参数验证
	if req.SenderId == "" {
		return nil, status.Error(codes.InvalidArgument, "sender_id is required")
	}
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}
	if req.ContentType == "" {
		return nil, status.Error(codes.InvalidArgument, "content_type is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}
	if req.GetLocalId() == "" {
		return nil, status.Error(codes.InvalidArgument, "local_id is required")
	}

	resp, err := s.messageService.SendMessage(ctx, req)
	if err != nil {
		logger.Error("Failed to send message", zap.Error(err))
		return nil, toStatusError(err)
	}

	return resp, nil
}

// GetMessages 获取消息列表
func (s *Server) GetMessages(ctx context.Context, req *messagepb.GetMessagesRequest) (*messagepb.GetMessagesResponse, error) {
	logger.Info("GetMessages called",
		zap.String("conversationId", req.ConversationId),
		zap.Int32("limit", req.Limit))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}

	resp, err := s.messageService.GetMessages(ctx, req)
	if err != nil {
		logger.Error("Failed to get messages", zap.Error(err))
		return nil, toStatusError(err)
	}

	return resp, nil
}

// GetMessageById 根据ID获取消息
func (s *Server) GetMessageById(ctx context.Context, req *messagepb.GetMessageByIdRequest) (*messagepb.Message, error) {
	logger.Info("GetMessageById called", zap.String("messageId", req.MessageId))

	// 参数验证
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id is required")
	}

	msg, err := s.messageService.GetMessageById(ctx, req.MessageId)
	if err != nil {
		logger.Error("Failed to get message", zap.Error(err))
		return nil, toStatusError(err)
	}

	return msg, nil
}

// RecallMessage 撤回消息
func (s *Server) RecallMessage(ctx context.Context, req *messagepb.RecallMessageRequest) (*commonpb.Empty, error) {
	operatorUserID := getOperatorUserID(ctx)
	logger.Info("RecallMessage called",
		zap.String("messageId", req.MessageId),
		zap.String("userId", operatorUserID))

	// 参数验证
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id is required")
	}
	if operatorUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "x-user-id metadata is required")
	}

	err := s.messageService.RecallMessage(ctx, req.MessageId, operatorUserID)
	if err != nil {
		logger.Error("Failed to recall message", zap.Error(err))
		return nil, toStatusError(err)
	}

	return &commonpb.Empty{}, nil
}

// DeleteMessage 删除消息
func (s *Server) DeleteMessage(ctx context.Context, req *messagepb.DeleteMessageRequest) (*commonpb.Empty, error) {
	operatorUserID := getOperatorUserID(ctx)
	logger.Info("DeleteMessage called",
		zap.String("messageId", req.MessageId),
		zap.String("userId", operatorUserID))

	// 参数验证
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id is required")
	}
	if operatorUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "x-user-id metadata is required")
	}

	err := s.messageService.DeleteMessage(ctx, req.MessageId, operatorUserID)
	if err != nil {
		logger.Error("Failed to delete message", zap.Error(err))
		return nil, toStatusError(err)
	}

	return &commonpb.Empty{}, nil
}

func getOperatorUserID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(operatorUserIDMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// MarkAsRead 标记消息已读
func (s *Server) MarkAsRead(ctx context.Context, req *messagepb.MarkAsReadRequest) (*commonpb.Empty, error) {
	operatorUserID := getOperatorUserID(ctx)
	logger.Info("MarkAsRead called",
		zap.String("conversationId", req.ConversationId),
		zap.String("userId", operatorUserID),
		zap.Int64("lastReadSeq", req.LastReadSeq))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}
	if operatorUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "x-user-id metadata is required")
	}

	err := s.messageService.MarkAsRead(ctx, operatorUserID, req)
	if err != nil {
		logger.Error("Failed to mark as read", zap.Error(err))
		return nil, toStatusError(err)
	}

	return &commonpb.Empty{}, nil
}

// MarkMessagesRead 批量按消息ID标记已读
func (s *Server) MarkMessagesRead(ctx context.Context, req *messagepb.MarkMessagesReadRequest) (*messagepb.MarkMessagesReadResponse, error) {
	operatorUserID := getOperatorUserID(ctx)
	logger.Info("MarkMessagesRead called",
		zap.String("conversationId", req.ConversationId),
		zap.String("userId", operatorUserID),
		zap.Int("messageCount", len(req.MessageIds)))

	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}
	if operatorUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "x-user-id metadata is required")
	}
	if len(req.MessageIds) == 0 {
		return &messagepb.MarkMessagesReadResponse{}, nil
	}

	resp, err := s.messageService.MarkMessagesRead(ctx, operatorUserID, req)
	if err != nil {
		logger.Error("Failed to mark messages as read", zap.Error(err))
		return nil, toStatusError(err)
	}

	return resp, nil
}

// AckReadTriggers 阅后即焚阅读触发回执
func (s *Server) AckReadTriggers(ctx context.Context, req *messagepb.AckReadTriggersRequest) (*messagepb.AckReadTriggersResponse, error) {
	operatorUserID := getOperatorUserID(ctx)
	logger.Info("AckReadTriggers called",
		zap.String("userId", operatorUserID),
		zap.Int("eventCount", len(req.Events)))

	if operatorUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "x-user-id metadata is required")
	}
	if len(req.Events) == 0 {
		return &messagepb.AckReadTriggersResponse{}, nil
	}

	resp, err := s.messageService.AckReadTriggers(ctx, operatorUserID, req)
	if err != nil {
		logger.Error("Failed to ack read triggers", zap.Error(err))
		return nil, toStatusError(err)
	}

	return resp, nil
}

// GetUnreadCount 获取未读消息数
func (s *Server) GetUnreadCount(ctx context.Context, req *messagepb.GetUnreadCountRequest) (*messagepb.GetUnreadCountResponse, error) {
	operatorUserID := getOperatorUserID(ctx)
	logger.Info("GetUnreadCount called",
		zap.String("conversationId", req.ConversationId),
		zap.String("userId", operatorUserID))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}
	if operatorUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "x-user-id metadata is required")
	}

	resp, err := s.messageService.GetUnreadCount(ctx, req.ConversationId, operatorUserID, req.LastReadSeq)
	if err != nil {
		logger.Error("Failed to get unread count", zap.Error(err))
		return nil, toStatusError(err)
	}

	return resp, nil
}

// GetReadReceipts 获取已读回执
func (s *Server) GetReadReceipts(ctx context.Context, req *messagepb.GetReadReceiptsRequest) (*messagepb.GetReadReceiptsResponse, error) {
	operatorUserID := getOperatorUserID(ctx)
	logger.Info("GetReadReceipts called",
		zap.String("conversationId", req.ConversationId),
		zap.String("userId", operatorUserID))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}

	if operatorUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "x-user-id metadata is required")
	}

	resp, err := s.messageService.GetReadReceipts(ctx, req.ConversationId, operatorUserID)
	if err != nil {
		logger.Error("Failed to get read receipts", zap.Error(err))
		return nil, toStatusError(err)
	}

	return resp, nil
}

// GetConversationSequence 获取会话序列号
func (s *Server) GetConversationSequence(ctx context.Context, req *messagepb.GetConversationSequenceRequest) (*messagepb.GetConversationSequenceResponse, error) {
	logger.Info("GetConversationSequence called", zap.String("conversationId", req.ConversationId))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}

	seq, err := s.messageService.GetConversationSequence(ctx, req.ConversationId)
	if err != nil {
		logger.Error("Failed to get conversation sequence", zap.Error(err))
		return nil, toStatusError(err)
	}

	return &messagepb.GetConversationSequenceResponse{
		CurrentSeq: seq,
	}, nil
}

// SearchMessages 搜索消息
func (s *Server) SearchMessages(ctx context.Context, req *messagepb.SearchMessagesRequest) (*messagepb.SearchMessagesResponse, error) {
	operatorUserID := getOperatorUserID(ctx)
	logger.Info("SearchMessages called",
		zap.String("userId", operatorUserID),
		zap.String("keyword", req.Keyword))

	// 参数验证
	if operatorUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "x-user-id metadata is required")
	}
	if req.Keyword == "" {
		return nil, status.Error(codes.InvalidArgument, "keyword is required")
	}
	if req.ConversationId == nil || *req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}

	resp, err := s.messageService.SearchMessages(ctx, operatorUserID, req)
	if err != nil {
		logger.Error("Failed to search messages", zap.Error(err))
		return nil, toStatusError(err)
	}

	return resp, nil
}

func toStatusError(err error) error {
	var bizErr *pkgerrors.Business
	if !stderrors.As(err, &bizErr) {
		return status.Error(codes.Internal, err.Error())
	}

	switch bizErr.Code {
	case pkgerrors.CodeParamError:
		return status.Error(codes.InvalidArgument, bizErr.Message)
	case pkgerrors.CodeConversationNotFound, pkgerrors.CodeMessageNotFound:
		return status.Error(codes.NotFound, bizErr.Message)
	case pkgerrors.CodeMessagePermissionDenied:
		return status.Error(codes.PermissionDenied, bizErr.Message)
	default:
		return status.Error(codes.Internal, bizErr.Message)
	}
}
