package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	lksdk "github.com/livekit/server-sdk-go"
	"github.com/livekit/protocol/auth"
	lkproto "github.com/livekit/protocol/livekit"

	rtcpb "github.com/anychat/server/api/proto/rtc"
	"github.com/anychat/server/internal/rtc/model"
	"github.com/anychat/server/internal/rtc/repository"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const tokenTTL = 2 * time.Hour

// RTCService RTC 音视频服务接口
type RTCService interface {
	InitiateCall(ctx context.Context, callerID, calleeID, callType string) (*rtcpb.InitiateCallResponse, error)
	JoinCall(ctx context.Context, callID, userID string) (*rtcpb.JoinCallResponse, error)
	RejectCall(ctx context.Context, callID, userID string) error
	EndCall(ctx context.Context, callID, userID string) error
	GetCallSession(ctx context.Context, callID, userID string) (*rtcpb.CallSession, error)
	ListCallLogs(ctx context.Context, userID string, page, pageSize int) (*rtcpb.ListCallLogsResponse, error)
	CreateMeeting(ctx context.Context, creatorID, title, password string, maxParticipants int) (*rtcpb.CreateMeetingResponse, error)
	JoinMeeting(ctx context.Context, userID, roomID, password string) (*rtcpb.JoinMeetingResponse, error)
	EndMeeting(ctx context.Context, roomID, creatorID string) error
	GetMeeting(ctx context.Context, roomID string) (*rtcpb.MeetingRoom, error)
	ListMeetings(ctx context.Context, page, pageSize int) (*rtcpb.ListMeetingsResponse, error)
}

type rtcServiceImpl struct {
	apiKey          string
	apiSecret       string
	serverURL       string
	roomClient      *lksdk.RoomServiceClient
	callRepo        repository.CallRepository
	meetingRepo     repository.MeetingRepository
	notificationPub notification.Publisher
}

// NewRTCService 创建音视频服务
func NewRTCService(
	serverURL, apiKey, apiSecret string,
	callRepo repository.CallRepository,
	meetingRepo repository.MeetingRepository,
	notificationPub notification.Publisher,
) RTCService {
	roomClient := lksdk.NewRoomServiceClient(serverURL, apiKey, apiSecret)
	return &rtcServiceImpl{
		apiKey:          apiKey,
		apiSecret:       apiSecret,
		serverURL:       serverURL,
		roomClient:      roomClient,
		callRepo:        callRepo,
		meetingRepo:     meetingRepo,
		notificationPub: notificationPub,
	}
}

// ── 通话相关 ──────────────────────────────────────────────

func (s *rtcServiceImpl) InitiateCall(ctx context.Context, callerID, calleeID, callType string) (*rtcpb.InitiateCallResponse, error) {
	callID := uuid.NewString()
	roomName := "call_" + callID

	// 创建 LiveKit Room（设置 EmptyTimeout 为5分钟，等待被叫接听）
	emptyTimeout := uint32(300)
	_, err := s.roomClient.CreateRoom(ctx, &lkproto.CreateRoomRequest{
		Name:         roomName,
		EmptyTimeout: emptyTimeout,
	})
	if err != nil {
		logger.Error("InitiateCall: create livekit room failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "create room: %v", err)
	}

	// 生成主叫 Token
	token, err := s.generateToken(roomName, callerID, true)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate token: %v", err)
	}

	// 持久化通话会话
	session := &model.CallSession{
		CallID:   callID,
		CallerID: callerID,
		CalleeID: calleeID,
		CallType: callType,
		Status:   "ringing",
		RoomName: roomName,
	}
	if err := s.callRepo.CreateCallSession(session); err != nil {
		logger.Error("InitiateCall: save session failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "save session: %v", err)
	}

	// 通知被叫方
	notif := notification.NewNotification(notification.TypeLiveKitCallInvite, callerID, notification.PriorityHigh).
		AddPayloadField("call_id", callID).
		AddPayloadField("caller_id", callerID).
		AddPayloadField("call_type", callType)
	if err := s.notificationPub.PublishToUser(calleeID, notif); err != nil {
		logger.Warn("InitiateCall: notify callee failed", zap.Error(err))
	}

	return &rtcpb.InitiateCallResponse{
		CallId:   callID,
		RoomName: roomName,
		Token:    token,
	}, nil
}

func (s *rtcServiceImpl) JoinCall(ctx context.Context, callID, userID string) (*rtcpb.JoinCallResponse, error) {
	session, err := s.callRepo.GetCallSession(callID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "call session not found")
	}
	if session.Status != "ringing" {
		return nil, status.Errorf(codes.FailedPrecondition, "call is not in ringing state: %s", session.Status)
	}
	if session.CalleeID != userID {
		return nil, status.Error(codes.PermissionDenied, "not the callee of this call")
	}

	// 生成被叫 Token
	token, err := s.generateToken(session.RoomName, userID, false)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate token: %v", err)
	}

	// 更新会话状态
	now := time.Now()
	session.Status = "connected"
	session.ConnectedAt = &now
	if err := s.callRepo.UpdateCallSession(session); err != nil {
		logger.Warn("JoinCall: update session failed", zap.Error(err))
	}

	// 通知主叫方
	notif := notification.NewNotification(notification.TypeLiveKitCallStatus, userID, notification.PriorityHigh).
		AddPayloadField("call_id", callID).
		AddPayloadField("status", "connected")
	if err := s.notificationPub.PublishToUser(session.CallerID, notif); err != nil {
		logger.Warn("JoinCall: notify caller failed", zap.Error(err))
	}

	return &rtcpb.JoinCallResponse{
		RoomName: session.RoomName,
		Token:    token,
	}, nil
}

func (s *rtcServiceImpl) RejectCall(ctx context.Context, callID, userID string) error {
	session, err := s.callRepo.GetCallSession(callID)
	if err != nil {
		return status.Error(codes.NotFound, "call session not found")
	}
	if session.Status != "ringing" {
		return status.Errorf(codes.FailedPrecondition, "call is not in ringing state: %s", session.Status)
	}
	if session.CalleeID != userID {
		return status.Error(codes.PermissionDenied, "not the callee of this call")
	}

	now := time.Now()
	session.Status = "rejected"
	session.EndedAt = &now
	if err := s.callRepo.UpdateCallSession(session); err != nil {
		logger.Warn("RejectCall: update session failed", zap.Error(err))
	}

	// 删除 Room（无需等待）
	go s.deleteRoom(session.RoomName)

	// 通知主叫方
	notif := notification.NewNotification(notification.TypeLiveKitCallRejected, userID, notification.PriorityHigh).
		AddPayloadField("call_id", callID).
		AddPayloadField("callee_id", userID)
	if err := s.notificationPub.PublishToUser(session.CallerID, notif); err != nil {
		logger.Warn("RejectCall: notify caller failed", zap.Error(err))
	}
	return nil
}

func (s *rtcServiceImpl) EndCall(ctx context.Context, callID, userID string) error {
	session, err := s.callRepo.GetCallSession(callID)
	if err != nil {
		return status.Error(codes.NotFound, "call session not found")
	}
	if session.CallerID != userID && session.CalleeID != userID {
		return status.Error(codes.PermissionDenied, "not a participant of this call")
	}
	if session.Status == "ended" || session.Status == "rejected" {
		return nil
	}

	now := time.Now()
	newStatus := "ended"
	if session.Status == "ringing" {
		newStatus = "cancelled"
	}
	session.Status = newStatus
	session.EndedAt = &now
	if session.ConnectedAt != nil {
		session.Duration = int(now.Sub(*session.ConnectedAt).Seconds())
	}
	if err := s.callRepo.UpdateCallSession(session); err != nil {
		logger.Warn("EndCall: update session failed", zap.Error(err))
	}

	go s.deleteRoom(session.RoomName)

	// 通知对方
	targetID := session.CallerID
	if userID == session.CallerID {
		targetID = session.CalleeID
	}
	notif := notification.NewNotification(notification.TypeLiveKitCallStatus, userID, notification.PriorityHigh).
		AddPayloadField("call_id", callID).
		AddPayloadField("status", newStatus)
	if err := s.notificationPub.PublishToUser(targetID, notif); err != nil {
		logger.Warn("EndCall: notify peer failed", zap.Error(err))
	}
	return nil
}

func (s *rtcServiceImpl) GetCallSession(ctx context.Context, callID, userID string) (*rtcpb.CallSession, error) {
	session, err := s.callRepo.GetCallSession(callID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "call session not found")
	}
	if session.CallerID != userID && session.CalleeID != userID {
		return nil, status.Error(codes.PermissionDenied, "not a participant of this call")
	}
	return toProtoCallSession(session), nil
}

func (s *rtcServiceImpl) ListCallLogs(ctx context.Context, userID string, page, pageSize int) (*rtcpb.ListCallLogsResponse, error) {
	sessions, total, err := s.callRepo.ListCallLogs(userID, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list call logs: %v", err)
	}
	pbSessions := make([]*rtcpb.CallSession, len(sessions))
	for i, s := range sessions {
		pbSessions[i] = toProtoCallSession(s)
	}
	return &rtcpb.ListCallLogsResponse{Sessions: pbSessions, Total: total}, nil
}

// ── 会议室相关 ────────────────────────────────────────────

func (s *rtcServiceImpl) CreateMeeting(ctx context.Context, creatorID, title, password string, maxParticipants int) (*rtcpb.CreateMeetingResponse, error) {
	roomID := uuid.NewString()
	roomName := "meeting_" + roomID

	_, err := s.roomClient.CreateRoom(ctx, &lkproto.CreateRoomRequest{
		Name:            roomName,
		MaxParticipants: uint32(maxParticipants),
	})
	if err != nil {
		logger.Error("CreateMeeting: create livekit room failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "create room: %v", err)
	}

	var passwordHash string
	if password != "" {
		passwordHash = hashPassword(password)
	}

	meeting := &model.MeetingRoom{
		RoomID:          roomID,
		CreatorID:       creatorID,
		Title:           title,
		RoomName:        roomName,
		PasswordHash:    passwordHash,
		MaxParticipants: maxParticipants,
		Status:          "active",
	}
	if err := s.meetingRepo.CreateMeeting(meeting); err != nil {
		return nil, status.Errorf(codes.Internal, "save meeting: %v", err)
	}

	// 创建者拥有 RoomAdmin 权限
	token, err := s.generateToken(roomName, creatorID, true)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate token: %v", err)
	}

	return &rtcpb.CreateMeetingResponse{
		Meeting: toProtoMeeting(meeting),
		Token:   token,
	}, nil
}

func (s *rtcServiceImpl) JoinMeeting(ctx context.Context, userID, roomID, password string) (*rtcpb.JoinMeetingResponse, error) {
	meeting, err := s.meetingRepo.GetMeetingByRoomID(roomID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "meeting not found")
	}
	if meeting.Status != "active" {
		return nil, status.Error(codes.FailedPrecondition, "meeting has ended")
	}
	if meeting.PasswordHash != "" && hashPassword(password) != meeting.PasswordHash {
		return nil, status.Error(codes.PermissionDenied, "incorrect password")
	}

	isAdmin := meeting.CreatorID == userID
	token, err := s.generateToken(meeting.RoomName, userID, isAdmin)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate token: %v", err)
	}
	return &rtcpb.JoinMeetingResponse{
		Meeting: toProtoMeeting(meeting),
		Token:   token,
	}, nil
}

func (s *rtcServiceImpl) EndMeeting(ctx context.Context, roomID, creatorID string) error {
	meeting, err := s.meetingRepo.GetMeetingByRoomID(roomID)
	if err != nil {
		return status.Error(codes.NotFound, "meeting not found")
	}
	if meeting.CreatorID != creatorID {
		return status.Error(codes.PermissionDenied, "only creator can end the meeting")
	}
	if meeting.Status == "ended" {
		return nil
	}

	now := time.Now()
	meeting.Status = "ended"
	meeting.EndedAt = &now
	if err := s.meetingRepo.UpdateMeeting(meeting); err != nil {
		logger.Warn("EndMeeting: update meeting failed", zap.Error(err))
	}

	go s.deleteRoom(meeting.RoomName)
	return nil
}

func (s *rtcServiceImpl) GetMeeting(ctx context.Context, roomID string) (*rtcpb.MeetingRoom, error) {
	meeting, err := s.meetingRepo.GetMeetingByRoomID(roomID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "meeting not found")
	}
	return toProtoMeeting(meeting), nil
}

func (s *rtcServiceImpl) ListMeetings(ctx context.Context, page, pageSize int) (*rtcpb.ListMeetingsResponse, error) {
	meetings, total, err := s.meetingRepo.ListActiveMeetings(page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list meetings: %v", err)
	}
	pbMeetings := make([]*rtcpb.MeetingRoom, len(meetings))
	for i, m := range meetings {
		pbMeetings[i] = toProtoMeeting(m)
	}
	return &rtcpb.ListMeetingsResponse{Meetings: pbMeetings, Total: total}, nil
}

// ── 内部辅助 ──────────────────────────────────────────────

// generateToken 生成 LiveKit JWT
// isAdmin=true 时附加 RoomAdmin 权限（会议室创建者/通话主叫方）
func (s *rtcServiceImpl) generateToken(roomName, identity string, isAdmin bool) (string, error) {
	at := auth.NewAccessToken(s.apiKey, s.apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin:  true,
		Room:      roomName,
		RoomAdmin: isAdmin,
	}
	at.AddGrant(grant).
		SetIdentity(identity).
		SetValidFor(tokenTTL)
	return at.ToJWT()
}

func (s *rtcServiceImpl) deleteRoom(roomName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := s.roomClient.DeleteRoom(ctx, &lkproto.DeleteRoomRequest{Room: roomName}); err != nil {
		logger.Warn("deleteRoom: failed", zap.String("room", roomName), zap.Error(err))
	}
}

func hashPassword(password string) string {
	h := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", h)
}

// ── Proto 转换 ────────────────────────────────────────────

func toProtoCallSession(s *model.CallSession) *rtcpb.CallSession {
	pb := &rtcpb.CallSession{
		CallId:    s.CallID,
		CallerId:  s.CallerID,
		CalleeId:  s.CalleeID,
		RoomName:  s.RoomName,
		StartedAt: s.StartedAt.Unix(),
		Duration:  int32(s.Duration),
		CreatedAt: s.CreatedAt.Unix(),
	}

	switch s.CallType {
	case "video":
		pb.CallType = rtcpb.CallType_CALL_TYPE_VIDEO
	default:
		pb.CallType = rtcpb.CallType_CALL_TYPE_AUDIO
	}

	switch s.Status {
	case "connected":
		pb.Status = rtcpb.CallStatus_CALL_STATUS_CONNECTED
	case "ended":
		pb.Status = rtcpb.CallStatus_CALL_STATUS_ENDED
	case "rejected":
		pb.Status = rtcpb.CallStatus_CALL_STATUS_REJECTED
	case "missed":
		pb.Status = rtcpb.CallStatus_CALL_STATUS_MISSED
	case "cancelled":
		pb.Status = rtcpb.CallStatus_CALL_STATUS_CANCELLED
	default:
		pb.Status = rtcpb.CallStatus_CALL_STATUS_RINGING
	}

	if s.ConnectedAt != nil {
		pb.ConnectedAt = s.ConnectedAt.Unix()
	}
	if s.EndedAt != nil {
		pb.EndedAt = s.EndedAt.Unix()
	}
	return pb
}

func toProtoMeeting(m *model.MeetingRoom) *rtcpb.MeetingRoom {
	pb := &rtcpb.MeetingRoom{
		RoomId:          m.RoomID,
		CreatorId:       m.CreatorID,
		Title:           m.Title,
		RoomName:        m.RoomName,
		HasPassword:     m.PasswordHash != "",
		MaxParticipants: int32(m.MaxParticipants),
		StartedAt:       m.StartedAt.Unix(),
		CreatedAt:       m.CreatedAt.Unix(),
	}
	if m.Status == "ended" {
		pb.Status = rtcpb.MeetingStatus_MEETING_STATUS_ENDED
	}
	if m.EndedAt != nil {
		pb.EndedAt = m.EndedAt.Unix()
	}
	return pb
}
