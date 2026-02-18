package grpc

import (
	"context"

	pushpb "github.com/anychat/server/api/proto/push"
	"github.com/anychat/server/internal/push/service"
	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server Push gRPC 服务器
type Server struct {
	pushpb.UnimplementedPushServiceServer
	pushService service.PushService
}

// NewServer 创建 gRPC 服务器
func NewServer(pushService service.PushService) *Server {
	return &Server{pushService: pushService}
}

// SendPush 向指定用户列表发送推送通知
func (s *Server) SendPush(ctx context.Context, req *pushpb.SendPushRequest) (*pushpb.SendPushResponse, error) {
	if len(req.UserIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_ids is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}

	successCount, failureCount, msgID, err := s.pushService.SendPush(
		ctx,
		req.UserIds,
		req.Title,
		req.Content,
		req.PushType,
		req.Extras,
	)
	if err != nil {
		logger.Error("SendPush gRPC failed", zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pushpb.SendPushResponse{
		SuccessCount: int32(successCount),
		FailureCount: int32(failureCount),
		MsgId:        msgID,
	}, nil
}
