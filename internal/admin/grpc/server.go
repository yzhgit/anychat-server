package grpc

import (
	"context"

	adminpb "github.com/anychat/server/api/proto/admin"
	commonpb "github.com/anychat/server/api/proto/common"
	"github.com/anychat/server/internal/admin/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server Admin gRPC 服务器
type Server struct {
	adminpb.UnimplementedAdminServiceServer
	svc service.AdminService
}

// NewServer 创建 gRPC 服务器
func NewServer(svc service.AdminService) *Server {
	return &Server{svc: svc}
}

func (s *Server) GetSystemStats(ctx context.Context, _ *adminpb.GetSystemStatsRequest) (*adminpb.GetSystemStatsResponse, error) {
	return s.svc.GetSystemStats(ctx)
}

func (s *Server) BanUser(ctx context.Context, req *adminpb.BanUserRequest) (*commonpb.Empty, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if err := s.svc.BanUser(ctx, "", req.UserId, req.Reason); err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}

func (s *Server) UnbanUser(ctx context.Context, req *adminpb.UnbanUserRequest) (*commonpb.Empty, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if err := s.svc.UnbanUser(ctx, "", req.UserId); err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}
