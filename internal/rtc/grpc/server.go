package grpc

import (
	"context"

	commonpb "github.com/anychat/server/api/proto/common"
	rtcpb "github.com/anychat/server/api/proto/rtc"
	"github.com/anychat/server/internal/rtc/service"
	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server RTC gRPC 服务器（实现 RTCService 接口）
type Server struct {
	rtcpb.UnimplementedRTCServiceServer
	svc service.RTCService
}

// NewServer 创建 gRPC 服务器
func NewServer(svc service.RTCService) *Server {
	return &Server{svc: svc}
}

func (s *Server) InitiateCall(ctx context.Context, req *rtcpb.InitiateCallRequest) (*rtcpb.InitiateCallResponse, error) {
	if req.CallerId == "" || req.CalleeId == "" {
		return nil, status.Error(codes.InvalidArgument, "caller_id and callee_id are required")
	}
	if req.CallerId == req.CalleeId {
		return nil, status.Error(codes.InvalidArgument, "caller and callee cannot be the same")
	}
	callType := "audio"
	if req.CallType == rtcpb.CallType_CALL_TYPE_VIDEO {
		callType = "video"
	}
	resp, err := s.svc.InitiateCall(ctx, req.CallerId, req.CalleeId, callType)
	if err != nil {
		logger.Error("InitiateCall gRPC failed", zap.Error(err))
		return nil, err
	}
	return resp, nil
}

func (s *Server) JoinCall(ctx context.Context, req *rtcpb.JoinCallRequest) (*rtcpb.JoinCallResponse, error) {
	if req.CallId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "call_id and user_id are required")
	}
	return s.svc.JoinCall(ctx, req.CallId, req.UserId)
}

func (s *Server) RejectCall(ctx context.Context, req *rtcpb.RejectCallRequest) (*commonpb.Empty, error) {
	if req.CallId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "call_id and user_id are required")
	}
	if err := s.svc.RejectCall(ctx, req.CallId, req.UserId); err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}

func (s *Server) EndCall(ctx context.Context, req *rtcpb.EndCallRequest) (*commonpb.Empty, error) {
	if req.CallId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "call_id and user_id are required")
	}
	if err := s.svc.EndCall(ctx, req.CallId, req.UserId); err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}

func (s *Server) GetCallSession(ctx context.Context, req *rtcpb.GetCallSessionRequest) (*rtcpb.CallSession, error) {
	if req.CallId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "call_id and user_id are required")
	}
	return s.svc.GetCallSession(ctx, req.CallId, req.UserId)
}

func (s *Server) ListCallLogs(ctx context.Context, req *rtcpb.ListCallLogsRequest) (*rtcpb.ListCallLogsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	return s.svc.ListCallLogs(ctx, req.UserId, int(req.Page), int(req.PageSize))
}

func (s *Server) CreateMeeting(ctx context.Context, req *rtcpb.CreateMeetingRequest) (*rtcpb.CreateMeetingResponse, error) {
	if req.CreatorId == "" || req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "creator_id and title are required")
	}
	return s.svc.CreateMeeting(ctx, req.CreatorId, req.Title, req.Password, int(req.MaxParticipants))
}

func (s *Server) JoinMeeting(ctx context.Context, req *rtcpb.JoinMeetingRequest) (*rtcpb.JoinMeetingResponse, error) {
	if req.UserId == "" || req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and room_id are required")
	}
	return s.svc.JoinMeeting(ctx, req.UserId, req.RoomId, req.Password)
}

func (s *Server) EndMeeting(ctx context.Context, req *rtcpb.EndMeetingRequest) (*commonpb.Empty, error) {
	if req.RoomId == "" || req.CreatorId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id and creator_id are required")
	}
	if err := s.svc.EndMeeting(ctx, req.RoomId, req.CreatorId); err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}

func (s *Server) GetMeeting(ctx context.Context, req *rtcpb.GetMeetingRequest) (*rtcpb.MeetingRoom, error) {
	if req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id is required")
	}
	return s.svc.GetMeeting(ctx, req.RoomId)
}

func (s *Server) ListMeetings(ctx context.Context, req *rtcpb.ListMeetingsRequest) (*rtcpb.ListMeetingsResponse, error) {
	return s.svc.ListMeetings(ctx, int(req.Page), int(req.PageSize))
}
