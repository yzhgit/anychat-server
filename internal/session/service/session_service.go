package service

import (
	"context"
	"fmt"
	"time"

	sessionpb "github.com/anychat/server/api/proto/session"
	"github.com/anychat/server/internal/session/model"
	"github.com/anychat/server/internal/session/repository"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// SessionService 会话服务接口
type SessionService interface {
	GetSessions(ctx context.Context, req *sessionpb.GetSessionsRequest) (*sessionpb.GetSessionsResponse, error)
	GetSession(ctx context.Context, userID, sessionID string) (*sessionpb.Session, error)
	CreateOrUpdateSession(ctx context.Context, req *sessionpb.CreateOrUpdateSessionRequest) (*sessionpb.Session, error)
	DeleteSession(ctx context.Context, userID, sessionID string) error
	SetPinned(ctx context.Context, userID, sessionID string, pinned bool) error
	SetMuted(ctx context.Context, userID, sessionID string, muted bool) error
	ClearUnread(ctx context.Context, userID, sessionID string) error
	GetTotalUnread(ctx context.Context, userID string) (int32, error)
	IncrUnread(ctx context.Context, userID, sessionID string, count int32) error
}

// sessionServiceImpl 会话服务实现
type sessionServiceImpl struct {
	sessionRepo     repository.SessionRepository
	notificationPub notification.Publisher
}

// NewSessionService 创建会话服务
func NewSessionService(
	sessionRepo repository.SessionRepository,
	notificationPub notification.Publisher,
) SessionService {
	return &sessionServiceImpl{
		sessionRepo:     sessionRepo,
		notificationPub: notificationPub,
	}
}

// GetSessions 获取用户会话列表
func (s *sessionServiceImpl) GetSessions(ctx context.Context, req *sessionpb.GetSessionsRequest) (*sessionpb.GetSessionsResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	var updatedBefore *time.Time
	if req.UpdatedBefore != nil {
		t := time.Unix(*req.UpdatedBefore, 0)
		updatedBefore = &t
	}

	sessions, err := s.sessionRepo.ListByUser(ctx, req.UserId, limit, updatedBefore)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	pbSessions := make([]*sessionpb.Session, 0, len(sessions))
	for _, s := range sessions {
		pbSessions = append(pbSessions, toProtoSession(s))
	}

	return &sessionpb.GetSessionsResponse{
		Sessions: pbSessions,
		HasMore:  len(sessions) == limit,
	}, nil
}

// GetSession 获取单个会话
func (s *sessionServiceImpl) GetSession(ctx context.Context, userID, sessionID string) (*sessionpb.Session, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("session not found")
	}
	return toProtoSession(session), nil
}

// CreateOrUpdateSession 创建或更新会话（消息到达时调用）
func (s *sessionServiceImpl) CreateOrUpdateSession(ctx context.Context, req *sessionpb.CreateOrUpdateSessionRequest) (*sessionpb.Session, error) {
	// 先尝试查找已有会话
	existing, err := s.sessionRepo.GetByUserAndTarget(ctx, req.UserId, req.SessionType, req.TargetId)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing session: %w", err)
	}

	var msgTime *time.Time
	if req.LastMessageTimestamp > 0 {
		t := time.Unix(req.LastMessageTimestamp, 0)
		msgTime = &t
	}

	if existing != nil {
		// 更新已有会话的最后消息信息
		existing.LastMessageID = req.LastMessageId
		existing.LastMessageContent = req.LastMessageContent
		existing.LastMessageTime = msgTime
		if err := s.sessionRepo.Upsert(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}
		return toProtoSession(existing), nil
	}

	// 创建新会话
	session := &model.Session{
		SessionID:          uuid.New().String(),
		SessionType:        req.SessionType,
		UserID:             req.UserId,
		TargetID:           req.TargetId,
		LastMessageID:      req.LastMessageId,
		LastMessageContent: req.LastMessageContent,
		LastMessageTime:    msgTime,
	}

	if err := s.sessionRepo.Upsert(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return toProtoSession(session), nil
}

// DeleteSession 删除会话并发送通知
func (s *sessionServiceImpl) DeleteSession(ctx context.Context, userID, sessionID string) error {
	if err := s.sessionRepo.Delete(ctx, userID, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// 发布会话删除通知（多端同步）
	notif := notification.NewNotification(notification.TypeSessionDeleted, userID, notification.PriorityNormal).
		AddPayloadField("session_id", sessionID)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish session deleted notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// SetPinned 设置置顶状态并发送通知
func (s *sessionServiceImpl) SetPinned(ctx context.Context, userID, sessionID string, pinned bool) error {
	var pinTime *time.Time
	if pinned {
		t := time.Now()
		pinTime = &t
	}

	if err := s.sessionRepo.SetPinned(ctx, userID, sessionID, pinned, pinTime); err != nil {
		return fmt.Errorf("failed to set pinned: %w", err)
	}

	// 发布置顶状态同步通知（多端同步）
	notif := notification.NewNotification(notification.TypeSessionPinUpdated, userID, notification.PriorityNormal).
		AddPayloadField("session_id", sessionID).
		AddPayloadField("is_pinned", pinned)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish session pin notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// SetMuted 设置免打扰状态并发送通知
func (s *sessionServiceImpl) SetMuted(ctx context.Context, userID, sessionID string, muted bool) error {
	if err := s.sessionRepo.SetMuted(ctx, userID, sessionID, muted); err != nil {
		return fmt.Errorf("failed to set muted: %w", err)
	}

	// 发布免打扰设置同步通知（多端同步）
	notif := notification.NewNotification(notification.TypeSessionMuteUpdated, userID, notification.PriorityNormal).
		AddPayloadField("session_id", sessionID).
		AddPayloadField("is_muted", muted)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish session mute notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// ClearUnread 清除未读数并发送通知
func (s *sessionServiceImpl) ClearUnread(ctx context.Context, userID, sessionID string) error {
	if err := s.sessionRepo.ClearUnread(ctx, userID, sessionID); err != nil {
		return fmt.Errorf("failed to clear unread: %w", err)
	}

	// 获取最新总未读数
	total, err := s.sessionRepo.SumUnread(ctx, userID)
	if err != nil {
		logger.Warn("Failed to get total unread after clear", zap.String("userID", userID), zap.Error(err))
	}

	// 发布未读数更新通知（多端同步）
	notif := notification.NewNotification(notification.TypeSessionUnreadUpdated, userID, notification.PriorityNormal).
		AddPayloadField("session_id", sessionID).
		AddPayloadField("unread_count", 0).
		AddPayloadField("total_unread", total)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish unread notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// GetTotalUnread 获取用户总未读数
func (s *sessionServiceImpl) GetTotalUnread(ctx context.Context, userID string) (int32, error) {
	total, err := s.sessionRepo.SumUnread(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get total unread: %w", err)
	}
	return total, nil
}

// IncrUnread 增加未读数并发送通知
func (s *sessionServiceImpl) IncrUnread(ctx context.Context, userID, sessionID string, count int32) error {
	if err := s.sessionRepo.IncrUnread(ctx, userID, sessionID, count); err != nil {
		return fmt.Errorf("failed to incr unread: %w", err)
	}

	// 发布未读数更新通知
	notif := notification.NewNotification(notification.TypeSessionUnreadUpdated, userID, notification.PriorityNormal).
		AddPayloadField("session_id", sessionID)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish unread incr notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// toProtoSession 将model.Session转换为protobuf Session
func toProtoSession(s *model.Session) *sessionpb.Session {
	pb := &sessionpb.Session{
		SessionId:          s.SessionID,
		SessionType:        s.SessionType,
		UserId:             s.UserID,
		TargetId:           s.TargetID,
		LastMessageId:      s.LastMessageID,
		LastMessageContent: s.LastMessageContent,
		UnreadCount:        s.UnreadCount,
		IsPinned:           s.IsPinned,
		IsMuted:            s.IsMuted,
		CreatedAt:          timestamppb.New(s.CreatedAt),
		UpdatedAt:          timestamppb.New(s.UpdatedAt),
	}
	if s.LastMessageTime != nil {
		pb.LastMessageTime = timestamppb.New(*s.LastMessageTime)
	}
	if s.PinTime != nil {
		pb.PinTime = timestamppb.New(*s.PinTime)
	}
	return pb
}
