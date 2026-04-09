package service

import (
	"context"
	"fmt"
	"time"

	conversationpb "github.com/anychat/server/api/proto/conversation"
	"github.com/anychat/server/internal/conversation/model"
	"github.com/anychat/server/internal/conversation/repository"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// ConversationService is the interface for conversation service
type ConversationService interface {
	GetConversations(ctx context.Context, req *conversationpb.GetConversationsRequest) (*conversationpb.GetConversationsResponse, error)
	GetConversation(ctx context.Context, userID, conversationID string) (*conversationpb.Conversation, error)
	CreateOrUpdateConversation(ctx context.Context, req *conversationpb.CreateOrUpdateConversationRequest) (*conversationpb.Conversation, error)
	DeleteConversation(ctx context.Context, userID, conversationID string) error
	SetPinned(ctx context.Context, userID, conversationID string, pinned bool) error
	SetMuted(ctx context.Context, userID, conversationID string, muted bool) error
	SetBurnAfterReading(ctx context.Context, userID, conversationID string, duration int32) error
	SetAutoDelete(ctx context.Context, userID, conversationID string, duration int32) error
	ClearUnread(ctx context.Context, userID, conversationID string) error
	GetTotalUnread(ctx context.Context, userID string) (int32, error)
	IncrUnread(ctx context.Context, userID, conversationID string, count int32) error
}

// conversationServiceImpl is the implementation of conversation service
type conversationServiceImpl struct {
	conversationRepo repository.ConversationRepository
	notificationPub  notification.Publisher
}

// NewConversationService creates a new conversation service
func NewConversationService(
	conversationRepo repository.ConversationRepository,
	notificationPub notification.Publisher,
) ConversationService {
	return &conversationServiceImpl{
		conversationRepo: conversationRepo,
		notificationPub:  notificationPub,
	}
}

// GetConversations retrieves the list of user conversations
func (s *conversationServiceImpl) GetConversations(ctx context.Context, req *conversationpb.GetConversationsRequest) (*conversationpb.GetConversationsResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	var updatedBefore *time.Time
	if req.UpdatedBefore != nil {
		t := time.Unix(*req.UpdatedBefore, 0)
		updatedBefore = &t
	}

	conversations, err := s.conversationRepo.ListByUser(ctx, req.UserId, limit, updatedBefore)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	pbConversations := make([]*conversationpb.Conversation, 0, len(conversations))
	for _, c := range conversations {
		pbConversations = append(pbConversations, toProtoConversation(c))
	}

	return &conversationpb.GetConversationsResponse{
		Conversations: pbConversations,
		HasMore:       len(conversations) == limit,
	}, nil
}

// GetConversation retrieves a single conversation
func (s *conversationServiceImpl) GetConversation(ctx context.Context, userID, conversationID string) (*conversationpb.Conversation, error) {
	conversation, err := s.conversationRepo.GetByID(ctx, conversationID)
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("conversation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	if conversation.UserID != userID {
		return nil, fmt.Errorf("conversation not found")
	}
	return toProtoConversation(conversation), nil
}

// CreateOrUpdateConversation creates or updates a conversation (called when message arrives)
func (s *conversationServiceImpl) CreateOrUpdateConversation(ctx context.Context, req *conversationpb.CreateOrUpdateConversationRequest) (*conversationpb.Conversation, error) {
	// Try to find existing conversation first
	existing, err := s.conversationRepo.GetByUserAndTarget(ctx, req.UserId, req.ConversationType, req.TargetId)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing conversation: %w", err)
	}

	var msgTime *time.Time
	if req.LastMessageTimestamp > 0 {
		t := time.Unix(req.LastMessageTimestamp, 0)
		msgTime = &t
	}

	if existing != nil {
		// Update existing conversation's last message info
		existing.LastMessageID = req.LastMessageId
		existing.LastMessageContent = req.LastMessageContent
		existing.LastMessageTime = msgTime
		if err := s.conversationRepo.Upsert(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to update conversation: %w", err)
		}
		return toProtoConversation(existing), nil
	}

	// Create new conversation
	conversation := &model.Conversation{
		ConversationID:     uuid.New().String(),
		ConversationType:   req.ConversationType,
		UserID:             req.UserId,
		TargetID:           req.TargetId,
		LastMessageID:      req.LastMessageId,
		LastMessageContent: req.LastMessageContent,
		LastMessageTime:    msgTime,
	}

	if err := s.conversationRepo.Upsert(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return toProtoConversation(conversation), nil
}

// DeleteConversation deletes a conversation and sends notification
func (s *conversationServiceImpl) DeleteConversation(ctx context.Context, userID, conversationID string) error {
	if err := s.conversationRepo.Delete(ctx, userID, conversationID); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	// Publish conversation deletion notification (multi-device sync)
	notif := notification.NewNotification(notification.TypeConversationDeleted, userID, notification.PriorityNormal).
		AddPayloadField("conversation_id", conversationID)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish conversation deleted notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// SetPinned sets pinned status and sends notification
func (s *conversationServiceImpl) SetPinned(ctx context.Context, userID, conversationID string, pinned bool) error {
	var pinTime *time.Time
	if pinned {
		t := time.Now()
		pinTime = &t
	}

	if err := s.conversationRepo.SetPinned(ctx, userID, conversationID, pinned, pinTime); err != nil {
		return fmt.Errorf("failed to set pinned: %w", err)
	}

	// Publish pinned status sync notification (multi-device sync)
	notif := notification.NewNotification(notification.TypeConversationPinUpdated, userID, notification.PriorityNormal).
		AddPayloadField("conversation_id", conversationID).
		AddPayloadField("is_pinned", pinned)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish conversation pin notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// SetMuted sets muted status and sends notification
func (s *conversationServiceImpl) SetMuted(ctx context.Context, userID, conversationID string, muted bool) error {
	if err := s.conversationRepo.SetMuted(ctx, userID, conversationID, muted); err != nil {
		return fmt.Errorf("failed to set muted: %w", err)
	}

	// Publish muted setting sync notification (multi-device sync)
	notif := notification.NewNotification(notification.TypeConversationMuteUpdated, userID, notification.PriorityNormal).
		AddPayloadField("conversation_id", conversationID).
		AddPayloadField("is_muted", muted)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish conversation mute notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// SetBurnAfterReading sets burn after reading duration and sends notification
func (s *conversationServiceImpl) SetBurnAfterReading(ctx context.Context, userID, conversationID string, duration int32) error {
	if err := s.conversationRepo.SetBurnAfterReading(ctx, userID, conversationID, duration); err != nil {
		return fmt.Errorf("failed to set burn after reading: %w", err)
	}

	// Publish burn after reading config change notification (multi-device sync)
	notif := notification.NewNotification(notification.TypeConversationBurnUpdated, userID, notification.PriorityNormal).
		AddPayloadField("conversation_id", conversationID).
		AddPayloadField("burn_after_reading", duration)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish conversation burn notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// SetAutoDelete sets auto delete duration and sends notification
func (s *conversationServiceImpl) SetAutoDelete(ctx context.Context, userID, conversationID string, duration int32) error {
	if err := s.conversationRepo.SetAutoDelete(ctx, userID, conversationID, duration); err != nil {
		return fmt.Errorf("failed to set auto delete: %w", err)
	}

	// Publish auto delete config change notification (multi-device sync)
	notif := notification.NewNotification(notification.TypeConversationAutoDeleteUpdated, userID, notification.PriorityNormal).
		AddPayloadField("conversation_id", conversationID).
		AddPayloadField("auto_delete_duration", duration)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish conversation auto delete notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// ClearUnread clears unread count and sends notification
func (s *conversationServiceImpl) ClearUnread(ctx context.Context, userID, conversationID string) error {
	if err := s.conversationRepo.ClearUnread(ctx, userID, conversationID); err != nil {
		return fmt.Errorf("failed to clear unread: %w", err)
	}

	// Get latest total unread count
	total, err := s.conversationRepo.SumUnread(ctx, userID)
	if err != nil {
		logger.Warn("Failed to get total unread after clear", zap.String("userID", userID), zap.Error(err))
	}

	// Publish unread count update notification (multi-device sync)
	notif := notification.NewNotification(notification.TypeConversationUnreadUpdated, userID, notification.PriorityNormal).
		AddPayloadField("conversation_id", conversationID).
		AddPayloadField("unread_count", 0).
		AddPayloadField("total_unread", total)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish unread notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// GetTotalUnread gets user's total unread count
func (s *conversationServiceImpl) GetTotalUnread(ctx context.Context, userID string) (int32, error) {
	total, err := s.conversationRepo.SumUnread(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get total unread: %w", err)
	}
	return total, nil
}

// IncrUnread increments unread count and sends notification
func (s *conversationServiceImpl) IncrUnread(ctx context.Context, userID, conversationID string, count int32) error {
	if err := s.conversationRepo.IncrUnread(ctx, userID, conversationID, count); err != nil {
		return fmt.Errorf("failed to incr conversation unread: %w", err)
	}

	// Publish unread count update notification
	notif := notification.NewNotification(notification.TypeConversationUnreadUpdated, userID, notification.PriorityNormal).
		AddPayloadField("conversation_id", conversationID)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Failed to publish unread incr notification",
			zap.String("userID", userID),
			zap.Error(err))
	}

	return nil
}

// toProtoConversation converts model.Conversation to protobuf Conversation
func toProtoConversation(s *model.Conversation) *conversationpb.Conversation {
	pb := &conversationpb.Conversation{
		ConversationId:     s.ConversationID,
		ConversationType:   s.ConversationType,
		UserId:             s.UserID,
		TargetId:           s.TargetID,
		LastMessageId:      s.LastMessageID,
		LastMessageContent: s.LastMessageContent,
		UnreadCount:        s.UnreadCount,
		IsPinned:           s.IsPinned,
		IsMuted:            s.IsMuted,
		BurnAfterReading:   s.BurnAfterReading,
		AutoDeleteDuration: s.AutoDeleteDuration,
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
