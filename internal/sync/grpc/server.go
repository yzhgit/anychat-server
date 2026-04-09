package grpc

import (
	"context"

	syncpb "github.com/anychat/server/api/proto/sync"
	"github.com/anychat/server/internal/sync/service"
	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server Sync gRPC server
type Server struct {
	syncpb.UnimplementedSyncServiceServer
	syncService service.SyncService
}

// NewServer creates gRPC server
func NewServer(syncService service.SyncService) *Server {
	return &Server{syncService: syncService}
}

// Sync full/incremental sync
func (s *Server) Sync(ctx context.Context, req *syncpb.SyncRequest) (*syncpb.SyncResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	resp, err := s.syncService.Sync(ctx, req)
	if err != nil {
		logger.Error("Sync failed", zap.String("userID", req.UserId), zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}

// SyncMessages message backfill
func (s *Server) SyncMessages(ctx context.Context, req *syncpb.SyncMessagesRequest) (*syncpb.SyncMessagesResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	resp, err := s.syncService.SyncMessages(ctx, req)
	if err != nil {
		logger.Error("SyncMessages failed", zap.String("userID", req.UserId), zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}
