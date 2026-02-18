package grpc

import (
	"context"

	commonpb "github.com/anychat/server/api/proto/common"
	messagepb "github.com/anychat/server/api/proto/message"
	"github.com/anychat/server/internal/message/service"
	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	if req.ConversationType == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_type is required")
	}
	if req.ContentType == "" {
		return nil, status.Error(codes.InvalidArgument, "content_type is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	resp, err := s.messageService.SendMessage(ctx, req)
	if err != nil {
		logger.Error("Failed to send message", zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
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
		return nil, status.Error(codes.Internal, err.Error())
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
		return nil, status.Error(codes.Internal, err.Error())
	}

	return msg, nil
}

// RecallMessage 撤回消息
func (s *Server) RecallMessage(ctx context.Context, req *messagepb.RecallMessageRequest) (*commonpb.Empty, error) {
	logger.Info("RecallMessage called",
		zap.String("messageId", req.MessageId),
		zap.String("userId", req.UserId))

	// 参数验证
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := s.messageService.RecallMessage(ctx, req.MessageId, req.UserId)
	if err != nil {
		logger.Error("Failed to recall message", zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &commonpb.Empty{}, nil
}

// DeleteMessage 删除消息
func (s *Server) DeleteMessage(ctx context.Context, req *messagepb.DeleteMessageRequest) (*commonpb.Empty, error) {
	logger.Info("DeleteMessage called",
		zap.String("messageId", req.MessageId),
		zap.String("userId", req.UserId))

	// 参数验证
	if req.MessageId == "" {
		return nil, status.Error(codes.InvalidArgument, "message_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := s.messageService.DeleteMessage(ctx, req.MessageId, req.UserId)
	if err != nil {
		logger.Error("Failed to delete message", zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &commonpb.Empty{}, nil
}

// MarkAsRead 标记消息已读
func (s *Server) MarkAsRead(ctx context.Context, req *messagepb.MarkAsReadRequest) (*commonpb.Empty, error) {
	logger.Info("MarkAsRead called",
		zap.String("conversationId", req.ConversationId),
		zap.String("userId", req.UserId),
		zap.Int64("lastReadSeq", req.LastReadSeq))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.ConversationType == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_type is required")
	}

	err := s.messageService.MarkAsRead(ctx, req)
	if err != nil {
		logger.Error("Failed to mark as read", zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &commonpb.Empty{}, nil
}

// GetUnreadCount 获取未读消息数
func (s *Server) GetUnreadCount(ctx context.Context, req *messagepb.GetUnreadCountRequest) (*messagepb.GetUnreadCountResponse, error) {
	logger.Info("GetUnreadCount called",
		zap.String("conversationId", req.ConversationId),
		zap.String("userId", req.UserId))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	resp, err := s.messageService.GetUnreadCount(ctx, req.ConversationId, req.UserId, req.LastReadSeq)
	if err != nil {
		logger.Error("Failed to get unread count", zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return resp, nil
}

// GetReadReceipts 获取已读回执
func (s *Server) GetReadReceipts(ctx context.Context, req *messagepb.GetReadReceiptsRequest) (*messagepb.GetReadReceiptsResponse, error) {
	logger.Info("GetReadReceipts called",
		zap.String("conversationId", req.ConversationId),
		zap.String("userId", req.UserId))

	// 参数验证
	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}

	resp, err := s.messageService.GetReadReceipts(ctx, req.ConversationId, req.UserId)
	if err != nil {
		logger.Error("Failed to get read receipts", zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
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
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &messagepb.GetConversationSequenceResponse{
		CurrentSeq: seq,
	}, nil
}

// SearchMessages 搜索消息
func (s *Server) SearchMessages(ctx context.Context, req *messagepb.SearchMessagesRequest) (*messagepb.SearchMessagesResponse, error) {
	logger.Info("SearchMessages called",
		zap.String("userId", req.UserId),
		zap.String("keyword", req.Keyword))

	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Keyword == "" {
		return nil, status.Error(codes.InvalidArgument, "keyword is required")
	}

	resp, err := s.messageService.SearchMessages(ctx, req)
	if err != nil {
		logger.Error("Failed to search messages", zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return resp, nil
}
