package grpc

import (
	"context"

	commonpb "github.com/anychat/server/api/proto/common"
	sessionpb "github.com/anychat/server/api/proto/session"
	"github.com/anychat/server/internal/session/service"
	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server Session gRPC服务器
type Server struct {
	sessionpb.UnimplementedSessionServiceServer
	sessionService service.SessionService
}

// NewServer 创建gRPC服务器
func NewServer(sessionService service.SessionService) *Server {
	return &Server{sessionService: sessionService}
}

// GetSessions 获取用户会话列表
func (s *Server) GetSessions(ctx context.Context, req *sessionpb.GetSessionsRequest) (*sessionpb.GetSessionsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	resp, err := s.sessionService.GetSessions(ctx, req)
	if err != nil {
		logger.Error("GetSessions failed", zap.String("userID", req.UserId), zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}

// GetSession 获取单个会话
func (s *Server) GetSession(ctx context.Context, req *sessionpb.GetSessionRequest) (*sessionpb.Session, error) {
	if req.UserId == "" || req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and session_id are required")
	}
	session, err := s.sessionService.GetSession(ctx, req.UserId, req.SessionId)
	if err != nil {
		logger.Error("GetSession failed", zap.String("sessionID", req.SessionId), zap.Error(err))
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return session, nil
}

// CreateOrUpdateSession 创建或更新会话
func (s *Server) CreateOrUpdateSession(ctx context.Context, req *sessionpb.CreateOrUpdateSessionRequest) (*sessionpb.Session, error) {
	if req.UserId == "" || req.TargetId == "" || req.SessionType == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, target_id and session_type are required")
	}
	session, err := s.sessionService.CreateOrUpdateSession(ctx, req)
	if err != nil {
		logger.Error("CreateOrUpdateSession failed", zap.String("userID", req.UserId), zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}
	return session, nil
}

// DeleteSession 删除会话
func (s *Server) DeleteSession(ctx context.Context, req *sessionpb.DeleteSessionRequest) (*commonpb.Empty, error) {
	if req.UserId == "" || req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and session_id are required")
	}
	if err := s.sessionService.DeleteSession(ctx, req.UserId, req.SessionId); err != nil {
		logger.Error("DeleteSession failed", zap.String("sessionID", req.SessionId), zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &commonpb.Empty{}, nil
}

// SetPinned 设置置顶
func (s *Server) SetPinned(ctx context.Context, req *sessionpb.SetPinnedRequest) (*commonpb.Empty, error) {
	if req.UserId == "" || req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and session_id are required")
	}
	if err := s.sessionService.SetPinned(ctx, req.UserId, req.SessionId, req.Pinned); err != nil {
		logger.Error("SetPinned failed", zap.String("sessionID", req.SessionId), zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &commonpb.Empty{}, nil
}

// SetMuted 设置免打扰
func (s *Server) SetMuted(ctx context.Context, req *sessionpb.SetMutedRequest) (*commonpb.Empty, error) {
	if req.UserId == "" || req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and session_id are required")
	}
	if err := s.sessionService.SetMuted(ctx, req.UserId, req.SessionId, req.Muted); err != nil {
		logger.Error("SetMuted failed", zap.String("sessionID", req.SessionId), zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &commonpb.Empty{}, nil
}

// ClearUnread 清除未读数
func (s *Server) ClearUnread(ctx context.Context, req *sessionpb.ClearUnreadRequest) (*commonpb.Empty, error) {
	if req.UserId == "" || req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and session_id are required")
	}
	if err := s.sessionService.ClearUnread(ctx, req.UserId, req.SessionId); err != nil {
		logger.Error("ClearUnread failed", zap.String("sessionID", req.SessionId), zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &commonpb.Empty{}, nil
}

// GetTotalUnread 获取总未读数
func (s *Server) GetTotalUnread(ctx context.Context, req *sessionpb.GetTotalUnreadRequest) (*sessionpb.GetTotalUnreadResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	total, err := s.sessionService.GetTotalUnread(ctx, req.UserId)
	if err != nil {
		logger.Error("GetTotalUnread failed", zap.String("userID", req.UserId), zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &sessionpb.GetTotalUnreadResponse{TotalUnread: total}, nil
}

// IncrUnread 增加未读数
func (s *Server) IncrUnread(ctx context.Context, req *sessionpb.IncrUnreadRequest) (*commonpb.Empty, error) {
	if req.UserId == "" || req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and session_id are required")
	}
	if err := s.sessionService.IncrUnread(ctx, req.UserId, req.SessionId, req.Count); err != nil {
		logger.Error("IncrUnread failed", zap.String("sessionID", req.SessionId), zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &commonpb.Empty{}, nil
}
