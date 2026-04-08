package service

import (
	"context"
	"encoding/json"
	"time"

	conversationpb "github.com/anychat/server/api/proto/conversation"
	grouppb "github.com/anychat/server/api/proto/group"
	messagepb "github.com/anychat/server/api/proto/message"
	"github.com/anychat/server/internal/message/model"
	"github.com/anychat/server/internal/message/repository"
	"github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// MessageService 消息服务接口
type MessageService interface {
	SendMessage(ctx context.Context, req *messagepb.SendMessageRequest) (*messagepb.SendMessageResponse, error)
	GetMessages(ctx context.Context, req *messagepb.GetMessagesRequest) (*messagepb.GetMessagesResponse, error)
	GetMessageById(ctx context.Context, messageID string) (*messagepb.Message, error)
	RecallMessage(ctx context.Context, messageID, userID string) error
	DeleteMessage(ctx context.Context, messageID, userID string) error
	MarkAsRead(ctx context.Context, userID string, req *messagepb.MarkAsReadRequest) error
	MarkMessagesRead(ctx context.Context, userID string, req *messagepb.MarkMessagesReadRequest) (*messagepb.MarkMessagesReadResponse, error)
	AckReadTriggers(ctx context.Context, userID string, req *messagepb.AckReadTriggersRequest) (*messagepb.AckReadTriggersResponse, error)
	GetUnreadCount(ctx context.Context, conversationID, userID string, lastReadSeq *int64) (*messagepb.GetUnreadCountResponse, error)
	GetReadReceipts(ctx context.Context, conversationID, userID string) (*messagepb.GetReadReceiptsResponse, error)
	GetConversationSequence(ctx context.Context, conversationID string) (int64, error)
	SearchMessages(ctx context.Context, userID string, req *messagepb.SearchMessagesRequest) (*messagepb.SearchMessagesResponse, error)
}

// MessageRepo 消息仓库接口
type MessageRepo interface {
	repository.MessageRepository
}

// ReadReceiptRepo 已读回执仓库接口
type ReadReceiptRepo interface {
	repository.ReadReceiptRepository
}

// SequenceRepo 序列号仓库接口
type SequenceRepo interface {
	repository.SequenceRepository
}

// SendIdempotencyRepo 发送幂等仓库接口
type SendIdempotencyRepo interface {
	repository.SendIdempotencyRepository
}

// messageServiceImpl 消息服务实现
type messageServiceImpl struct {
	messageRepo         MessageRepo
	readReceiptRepo     ReadReceiptRepo
	sequenceRepo        SequenceRepo
	sendIdempotencyRepo SendIdempotencyRepo
	conversationClient  conversationpb.ConversationServiceClient
	groupClient         grouppb.GroupServiceClient
	notificationPub     notification.Publisher
	db                  *gorm.DB
}

// NewMessageService 创建消息服务
func NewMessageService(
	messageRepo repository.MessageRepository,
	readReceiptRepo repository.ReadReceiptRepository,
	sequenceRepo repository.SequenceRepository,
	sendIdempotencyRepo repository.SendIdempotencyRepository,
	conversationClient conversationpb.ConversationServiceClient,
	groupClient grouppb.GroupServiceClient,
	notificationPub notification.Publisher,
	db *gorm.DB,
) MessageService {
	return &messageServiceImpl{
		messageRepo:         messageRepo,
		readReceiptRepo:     readReceiptRepo,
		sequenceRepo:        sequenceRepo,
		sendIdempotencyRepo: sendIdempotencyRepo,
		conversationClient:  conversationClient,
		groupClient:         groupClient,
		notificationPub:     notificationPub,
		db:                  db,
	}
}

// SendMessage 发送消息
func (s *messageServiceImpl) SendMessage(ctx context.Context, req *messagepb.SendMessageRequest) (*messagepb.SendMessageResponse, error) {
	conversation, err := s.authorizeSend(ctx, req.SenderId, req.ConversationId)
	if err != nil {
		return nil, err
	}

	localID := req.GetLocalId()
	if localID == "" {
		return nil, errors.NewBusiness(errors.CodeParamError, "local_id is required")
	}
	if s.sendIdempotencyRepo == nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "idempotency repo is not initialized")
	}

	var message *model.Message
	created := false

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		messageRepoTx := s.messageRepo.WithTx(tx)
		sequenceRepoTx := s.sequenceRepo.WithTx(tx)
		idempotencyRepoTx := s.sendIdempotencyRepo.WithTx(tx)

		if err := idempotencyRepoTx.CreateIfNotExists(ctx, &model.MessageSendIdempotency{
			SenderID:       req.SenderId,
			ConversationID: req.ConversationId,
			LocalID:        localID,
		}); err != nil {
			return err
		}

		idempotencyRecord, err := idempotencyRepoTx.GetForUpdateByKey(ctx, req.SenderId, req.ConversationId, localID)
		if err != nil {
			return err
		}

		// 幂等命中：直接返回既有消息
		if idempotencyRecord.MessageID != "" {
			existing, err := messageRepoTx.GetByMessageID(ctx, idempotencyRecord.MessageID)
			if err != nil {
				return err
			}
			message = existing
			return nil
		}

		// 创建新消息（与序列号分配在同一事务中）
		sequence, err := sequenceRepoTx.IncrementAndGet(ctx, req.ConversationId)
		if err != nil {
			logger.Error("Failed to increment sequence", zap.Error(err))
			return errors.NewBusiness(errors.CodeSequenceGenerateFailed, "")
		}

		now := time.Now()
		newMessage := &model.Message{
			MessageID:        uuid.New().String(),
			ConversationID:   req.ConversationId,
			ConversationType: conversation.ConversationType,
			TargetID:         conversation.TargetId,
			SenderID:         req.SenderId,
			ContentType:      req.ContentType,
			Content:          req.Content,
			Sequence:         sequence,
			Status:           model.MessageStatusNormal,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		if conversation.AutoDeleteDuration > 0 {
			expireTime := now.Add(time.Duration(conversation.AutoDeleteDuration) * time.Second)
			newMessage.AutoDeleteExpireTime = &expireTime
			newMessage.ExpireTime = &expireTime
		}
		if conversation.BurnAfterReading > 0 {
			newMessage.BurnAfterReadingSeconds = conversation.BurnAfterReading
		}
		if req.ReplyTo != nil {
			newMessage.ReplyTo = req.ReplyTo
		}
		if len(req.AtUsers) > 0 {
			newMessage.AtUsers = req.AtUsers
		}

		if err := messageRepoTx.Create(ctx, newMessage); err != nil {
			logger.Error("Failed to create message", zap.Error(err))
			return errors.NewBusiness(errors.CodeMessageSendFailed, "")
		}

		if err := idempotencyRepoTx.BindMessageID(ctx, req.SenderId, req.ConversationId, localID, newMessage.MessageID); err != nil {
			return err
		}

		message = newMessage
		created = true
		return nil
	})
	if err != nil {
		logger.Error("Failed to send message in transaction", zap.Error(err))
		if errors.IsBusiness(err) {
			return nil, err
		}
		return nil, errors.NewBusiness(errors.CodeMessageSendFailed, "")
	}

	if message == nil {
		return nil, errors.NewBusiness(errors.CodeMessageSendFailed, "")
	}

	if created {
		if err := s.publishNewMessageNotification(ctx, message, req.AtUsers); err != nil {
			logger.Error("Failed to publish message notification", zap.Error(err))
		}

		if len(req.AtUsers) > 0 {
			if err := s.publishMentionNotification(message); err != nil {
				logger.Error("Failed to publish mention notification", zap.Error(err))
			}
		}
	}

	return &messagepb.SendMessageResponse{
		MessageId: message.MessageID,
		Sequence:  message.Sequence,
		Timestamp: timestamppb.New(message.CreatedAt),
	}, nil
}

func (s *messageServiceImpl) authorizeSend(ctx context.Context, senderID, conversationID string) (*conversationpb.Conversation, error) {
	if s.conversationClient == nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "conversation client is not initialized")
	}
	if senderID == "" || conversationID == "" {
		return nil, errors.NewBusiness(errors.CodeParamError, "sender_id and conversation_id are required")
	}

	conversation, err := s.conversationClient.GetConversation(ctx, &conversationpb.GetConversationRequest{
		UserId:         senderID,
		ConversationId: conversationID,
	})
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeConversationNotFound, "conversation not found")
	}
	if conversation.ConversationType != model.ConversationTypeSingle && conversation.ConversationType != model.ConversationTypeGroup {
		return nil, errors.NewBusiness(errors.CodeParamError, "conversation_type must be single or group")
	}

	targetID := conversation.TargetId
	if targetID == "" {
		return nil, errors.NewBusiness(errors.CodeParamError, "target_id is required")
	}

	if conversation.ConversationType == model.ConversationTypeGroup {
		if s.groupClient == nil {
			return nil, errors.NewBusiness(errors.CodeInternalError, "group client is not initialized")
		}
		memberResp, err := s.groupClient.IsMember(ctx, &grouppb.IsMemberRequest{
			GroupId: targetID,
			UserId:  senderID,
		})
		if err != nil {
			return nil, errors.NewBusiness(errors.CodeInternalError, "failed to verify group membership")
		}
		if !memberResp.IsMember {
			return nil, errors.NewBusiness(errors.CodeMessagePermissionDenied, "sender is not a group member")
		}
	}

	return conversation, nil
}

// GetMessages 获取消息列表
func (s *messageServiceImpl) GetMessages(ctx context.Context, req *messagepb.GetMessagesRequest) (*messagepb.GetMessagesResponse, error) {
	// 参数验证
	if req.Limit <= 0 {
		req.Limit = 20 // 默认20条
	}
	if req.Limit > 100 {
		req.Limit = 100 // 最多100条
	}
	limit := int(req.Limit)

	// 获取消息列表
	startSeq := int64(0)
	endSeq := int64(0)
	if req.StartSeq != nil {
		startSeq = *req.StartSeq
	}
	if req.EndSeq != nil {
		endSeq = *req.EndSeq
	}

	messages, err := s.messageRepo.GetByConversation(ctx, req.ConversationId, startSeq, endSeq, limit+1, req.Reverse)
	if err != nil {
		logger.Error("Failed to get messages", zap.Error(err))
		return nil, errors.NewBusiness(errors.CodeInternalError, "Failed to retrieve messages")
	}

	hasMore := false
	if len(messages) > limit {
		hasMore = true
		messages = messages[:limit]
	}

	// 转换为protobuf消息
	pbMessages := make([]*messagepb.Message, 0, len(messages))
	for _, msg := range messages {
		pbMsg := s.modelToProto(msg)
		pbMessages = append(pbMessages, pbMsg)
	}

	return &messagepb.GetMessagesResponse{
		Messages: pbMessages,
		Total:    int64(len(messages)),
		HasMore:  hasMore,
	}, nil
}

// GetMessageById 根据ID获取消息
func (s *messageServiceImpl) GetMessageById(ctx context.Context, messageID string) (*messagepb.Message, error) {
	message, err := s.messageRepo.GetByMessageID(ctx, messageID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeMessageNotFound, "")
		}
		logger.Error("Failed to get message", zap.Error(err))
		return nil, errors.NewBusiness(errors.CodeInternalError, "Failed to retrieve message")
	}

	return s.modelToProto(message), nil
}

// RecallMessage 撤回消息
func (s *messageServiceImpl) RecallMessage(ctx context.Context, messageID, userID string) error {
	// 1. 获取消息
	message, err := s.messageRepo.GetByMessageID(ctx, messageID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeMessageNotFound, "")
		}
		return errors.NewBusiness(errors.CodeInternalError, "Failed to retrieve message")
	}

	// 2. 验证权限（只能撤回自己的消息）
	if message.SenderID != userID {
		return errors.NewBusiness(errors.CodeMessagePermissionDenied, "Cannot recall other's message")
	}

	// 3. 检查撤回时间限制（2分钟内）
	if time.Since(message.CreatedAt) > 2*time.Minute {
		return errors.NewBusiness(errors.CodeMessageRecallTimeLimit, "")
	}

	// 4. 更新消息状态为撤回
	if err := s.messageRepo.UpdateStatus(ctx, messageID, model.MessageStatusRecall); err != nil {
		logger.Error("Failed to recall message", zap.Error(err))
		return errors.NewBusiness(errors.CodeMessageRecallFailed, "")
	}

	// 5. 发布撤回通知
	if err := s.publishRecallNotification(ctx, message, userID); err != nil {
		logger.Error("Failed to publish recall notification", zap.Error(err))
	}

	return nil
}

// DeleteMessage 删除消息
func (s *messageServiceImpl) DeleteMessage(ctx context.Context, messageID, userID string) error {
	// 1. 获取消息
	message, err := s.messageRepo.GetByMessageID(ctx, messageID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeMessageNotFound, "")
		}
		return errors.NewBusiness(errors.CodeInternalError, "Failed to retrieve message")
	}

	// 2. 验证权限（只能删除自己的消息）
	if message.SenderID != userID {
		return errors.NewBusiness(errors.CodeMessagePermissionDenied, "Cannot delete other's message")
	}

	// 3. 软删除消息
	if err := s.messageRepo.Delete(ctx, messageID); err != nil {
		logger.Error("Failed to delete message", zap.Error(err))
		return errors.NewBusiness(errors.CodeMessageDeleteFailed, "")
	}

	return nil
}

// MarkAsRead 标记消息已读
func (s *messageServiceImpl) MarkAsRead(ctx context.Context, userID string, req *messagepb.MarkAsReadRequest) error {
	if s.conversationClient == nil {
		return errors.NewBusiness(errors.CodeInternalError, "conversation client is not initialized")
	}
	if userID == "" || req.ConversationId == "" {
		return errors.NewBusiness(errors.CodeParamError, "user_id and conversation_id are required")
	}

	conversation, err := s.conversationClient.GetConversation(ctx, &conversationpb.GetConversationRequest{
		UserId:         userID,
		ConversationId: req.ConversationId,
	})
	if err != nil {
		return errors.NewBusiness(errors.CodeConversationNotFound, "conversation not found")
	}
	if conversation.ConversationType != model.ConversationTypeSingle && conversation.ConversationType != model.ConversationTypeGroup {
		return errors.NewBusiness(errors.CodeParamError, "conversation_type must be single or group")
	}

	effectiveReadSeq := req.LastReadSeq
	lastReadMessageID := req.LastReadMessageId

	existingReceipt, err := s.readReceiptRepo.GetByConversationAndUser(ctx, req.ConversationId, userID)
	if err != nil && err != gorm.ErrRecordNotFound {
		logger.Error("Failed to get existing read receipt", zap.Error(err))
		return errors.NewBusiness(errors.CodeInternalError, "Failed to get read receipt")
	}
	if existingReceipt != nil && existingReceipt.LastReadSeq > effectiveReadSeq {
		effectiveReadSeq = existingReceipt.LastReadSeq
		lastReadMessageID = existingReceipt.LastReadMessageID
	}

	// 创建或更新已读回执
	receipt := &model.MessageReadReceipt{
		ConversationID:   req.ConversationId,
		ConversationType: conversation.ConversationType,
		TargetID:         conversation.TargetId,
		UserID:           userID,
		LastReadSeq:      effectiveReadSeq,
		ReadAt:           time.Now(),
	}

	if lastReadMessageID != nil {
		receipt.LastReadMessageID = lastReadMessageID
	}

	if err := s.readReceiptRepo.Upsert(ctx, receipt); err != nil {
		logger.Error("Failed to upsert read receipt", zap.Error(err))
		return errors.NewBusiness(errors.CodeMarkReadFailed, "")
	}

	// 发布已读回执通知（单聊时通知对方）
	if conversation.ConversationType == model.ConversationTypeSingle {
		if err := s.publishReadReceiptNotification(receipt); err != nil {
			logger.Error("Failed to publish read receipt notification", zap.Error(err))
		}
	}

	return nil
}

// MarkMessagesRead 批量按消息ID标记已读
func (s *messageServiceImpl) MarkMessagesRead(ctx context.Context, userID string, req *messagepb.MarkMessagesReadRequest) (*messagepb.MarkMessagesReadResponse, error) {
	if userID == "" {
		return nil, errors.NewBusiness(errors.CodeParamError, "user_id is required")
	}
	if req.ConversationId == "" {
		return nil, errors.NewBusiness(errors.CodeParamError, "conversation_id is required")
	}
	if len(req.MessageIds) == 0 {
		return &messagepb.MarkMessagesReadResponse{}, nil
	}
	if err := s.ensureConversationAccessible(ctx, userID, req.ConversationId); err != nil {
		return nil, err
	}

	messageSet := make(map[string]struct{}, len(req.MessageIds))
	messageIDs := make([]string, 0, len(req.MessageIds))
	for _, id := range req.MessageIds {
		if id == "" {
			continue
		}
		if _, exists := messageSet[id]; exists {
			continue
		}
		messageSet[id] = struct{}{}
		messageIDs = append(messageIDs, id)
	}
	if len(messageIDs) == 0 {
		return &messagepb.MarkMessagesReadResponse{}, nil
	}

	var messages []*model.Message
	if err := s.db.WithContext(ctx).
		Model(&model.Message{}).
		Select("message_id", "sequence", "conversation_id", "status").
		Where("conversation_id = ? AND message_id IN ? AND status = ?", req.ConversationId, messageIDs, model.MessageStatusNormal).
		Find(&messages).Error; err != nil {
		logger.Error("Failed to load messages for MarkMessagesRead", zap.Error(err))
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to mark messages as read")
	}

	maxSeq := int64(0)
	var maxSeqMessageID *string
	acceptedSet := make(map[string]struct{}, len(messages))
	for _, msg := range messages {
		acceptedSet[msg.MessageID] = struct{}{}
		if msg.Sequence >= maxSeq {
			maxSeq = msg.Sequence
			id := msg.MessageID
			maxSeqMessageID = &id
		}
	}

	acceptedIDs := make([]string, 0, len(acceptedSet))
	ignoredIDs := make([]string, 0, len(messageIDs))
	for _, id := range messageIDs {
		if _, ok := acceptedSet[id]; ok {
			acceptedIDs = append(acceptedIDs, id)
		} else {
			ignoredIDs = append(ignoredIDs, id)
		}
	}

	if len(acceptedIDs) == 0 {
		return &messagepb.MarkMessagesReadResponse{
			AcceptedIds:         acceptedIDs,
			IgnoredIds:          ignoredIDs,
			AdvancedLastReadSeq: 0,
		}, nil
	}

	currentSeq := int64(0)
	receipt, err := s.readReceiptRepo.GetByConversationAndUser(ctx, req.ConversationId, userID)
	if err != nil && err != gorm.ErrRecordNotFound {
		logger.Error("Failed to get read receipt before MarkMessagesRead", zap.Error(err))
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to mark messages as read")
	}
	if receipt != nil {
		currentSeq = receipt.LastReadSeq
	}

	advancedSeq := currentSeq
	if maxSeq > currentSeq {
		advancedSeq = maxSeq
		markReq := &messagepb.MarkAsReadRequest{
			ConversationId: req.ConversationId,
			LastReadSeq:    maxSeq,
		}
		if maxSeqMessageID != nil {
			markReq.LastReadMessageId = maxSeqMessageID
		}
		if err := s.MarkAsRead(ctx, userID, markReq); err != nil {
			return nil, err
		}
	}

	return &messagepb.MarkMessagesReadResponse{
		AcceptedIds:         acceptedIDs,
		IgnoredIds:          ignoredIDs,
		AdvancedLastReadSeq: advancedSeq,
	}, nil
}

// AckReadTriggers 批量上报阅后即焚阅读触发
func (s *messageServiceImpl) AckReadTriggers(ctx context.Context, userID string, req *messagepb.AckReadTriggersRequest) (*messagepb.AckReadTriggersResponse, error) {
	if userID == "" {
		return nil, errors.NewBusiness(errors.CodeParamError, "user_id is required")
	}
	if len(req.Events) == 0 {
		return &messagepb.AckReadTriggersResponse{}, nil
	}

	messageIDSet := make(map[string]struct{}, len(req.Events))
	messageIDs := make([]string, 0, len(req.Events))
	for _, event := range req.Events {
		if event.GetMessageId() == "" {
			continue
		}
		if _, exists := messageIDSet[event.GetMessageId()]; exists {
			continue
		}
		messageIDSet[event.GetMessageId()] = struct{}{}
		messageIDs = append(messageIDs, event.GetMessageId())
	}
	if len(messageIDs) == 0 {
		return &messagepb.AckReadTriggersResponse{}, nil
	}

	var candidates []*model.Message
	if err := s.db.WithContext(ctx).
		Model(&model.Message{}).
		Select("message_id", "sender_id", "burn_after_reading_seconds", "auto_delete_expire_time", "burn_after_reading_expire_time", "expire_time", "status").
		Where("message_id IN ? AND status = ? AND burn_after_reading_seconds > 0", messageIDs, model.MessageStatusNormal).
		Find(&candidates).Error; err != nil {
		logger.Error("Failed to load read trigger candidates", zap.Error(err))
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to ack read triggers")
	}

	now := time.Now()
	successIDs := make([]string, 0, len(candidates))
	ignoredSet := make(map[string]struct{}, len(messageIDs))
	for _, id := range messageIDs {
		ignoredSet[id] = struct{}{}
	}

	for _, msg := range candidates {
		// 发送方不能触发自己的阅后即焚
		if msg.SenderID == userID {
			continue
		}

		successIDs = append(successIDs, msg.MessageID)
		delete(ignoredSet, msg.MessageID)

		burnExpire := now.Add(time.Duration(msg.BurnAfterReadingSeconds) * time.Second)

		updates := map[string]interface{}{
			"updated_at": now,
		}
		shouldUpdate := false

		if msg.BurnAfterReadingExpireTime == nil || burnExpire.Before(*msg.BurnAfterReadingExpireTime) {
			updates["burn_after_reading_expire_time"] = burnExpire
			shouldUpdate = true
		}

		finalExpire := burnExpire
		if msg.AutoDeleteExpireTime != nil && msg.AutoDeleteExpireTime.Before(finalExpire) {
			finalExpire = *msg.AutoDeleteExpireTime
		}
		if msg.ExpireTime == nil || finalExpire.Before(*msg.ExpireTime) {
			updates["expire_time"] = finalExpire
			shouldUpdate = true
		}

		if !shouldUpdate {
			continue
		}

		if err := s.db.WithContext(ctx).
			Model(&model.Message{}).
			Where("message_id = ? AND status = ? AND sender_id <> ? AND burn_after_reading_seconds > 0", msg.MessageID, model.MessageStatusNormal, userID).
			Updates(updates).Error; err != nil {
			logger.Error("Failed to update burn expire time",
				zap.String("messageID", msg.MessageID),
				zap.Error(err))
			return nil, errors.NewBusiness(errors.CodeInternalError, "failed to ack read triggers")
		}
	}

	ignoredIDs := make([]string, 0, len(ignoredSet))
	for _, id := range messageIDs {
		if _, exists := ignoredSet[id]; exists {
			ignoredIDs = append(ignoredIDs, id)
		}
	}

	return &messagepb.AckReadTriggersResponse{
		SuccessIds: successIDs,
		IgnoredIds: ignoredIDs,
	}, nil
}

// GetUnreadCount 获取未读消息数
func (s *messageServiceImpl) GetUnreadCount(ctx context.Context, conversationID, userID string, lastReadSeq *int64) (*messagepb.GetUnreadCountResponse, error) {
	if err := s.ensureConversationAccessible(ctx, userID, conversationID); err != nil {
		return nil, err
	}

	// 如果没有提供lastReadSeq，从已读回执中获取
	var readSeq int64
	if lastReadSeq != nil {
		readSeq = *lastReadSeq
	} else {
		receipt, err := s.readReceiptRepo.GetByConversationAndUser(ctx, conversationID, userID)
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeInternalError, "Failed to get read receipt")
		}
		if receipt != nil {
			readSeq = receipt.LastReadSeq
		}
	}

	// 统计未读数
	unreadCount, err := s.messageRepo.CountUnreadByConversation(ctx, conversationID, readSeq)
	if err != nil {
		logger.Error("Failed to count unread messages", zap.Error(err))
		return nil, errors.NewBusiness(errors.CodeGetUnreadCountFailed, "")
	}

	// 获取最新消息序列号
	currentSeq, err := s.sequenceRepo.GetCurrentSeq(ctx, conversationID)
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "Failed to get current sequence")
	}

	// 获取最新一条消息
	var lastMessage *messagepb.Message
	messages, err := s.messageRepo.GetLatestByConversation(ctx, conversationID, 1)
	if err == nil && len(messages) > 0 {
		lastMessage = s.modelToProto(messages[0])
	}

	return &messagepb.GetUnreadCountResponse{
		UnreadCount:    unreadCount,
		LastMessageSeq: currentSeq,
		LastMessage:    lastMessage,
	}, nil
}

// GetReadReceipts 获取已读回执列表
func (s *messageServiceImpl) GetReadReceipts(ctx context.Context, conversationID, userID string) (*messagepb.GetReadReceiptsResponse, error) {
	if err := s.ensureConversationAccessible(ctx, userID, conversationID); err != nil {
		return nil, err
	}

	receipts, err := s.readReceiptRepo.GetByConversation(ctx, conversationID)
	if err != nil {
		logger.Error("Failed to get read receipts", zap.Error(err))
		return nil, errors.NewBusiness(errors.CodeInternalError, "Failed to retrieve read receipts")
	}

	pbReceipts := make([]*messagepb.ReadReceipt, 0, len(receipts))
	for _, receipt := range receipts {
		pbReceipt := &messagepb.ReadReceipt{
			UserId:      receipt.UserID,
			LastReadSeq: receipt.LastReadSeq,
			ReadAt:      timestamppb.New(receipt.ReadAt),
		}
		if receipt.LastReadMessageID != nil {
			pbReceipt.LastReadMessageId = receipt.LastReadMessageID
		}
		pbReceipts = append(pbReceipts, pbReceipt)
	}

	return &messagepb.GetReadReceiptsResponse{
		Receipts: pbReceipts,
	}, nil
}

// GetConversationSequence 获取会话当前序列号
func (s *messageServiceImpl) GetConversationSequence(ctx context.Context, conversationID string) (int64, error) {
	seq, err := s.sequenceRepo.GetCurrentSeq(ctx, conversationID)
	if err != nil {
		logger.Error("Failed to get conversation sequence", zap.Error(err))
		return 0, errors.NewBusiness(errors.CodeInternalError, "Failed to get sequence")
	}
	return seq, nil
}

// SearchMessages 搜索消息
func (s *messageServiceImpl) SearchMessages(ctx context.Context, userID string, req *messagepb.SearchMessagesRequest) (*messagepb.SearchMessagesResponse, error) {
	if userID == "" {
		return nil, errors.NewBusiness(errors.CodeParamError, "user_id is required")
	}
	if req.ConversationId == nil || *req.ConversationId == "" {
		return nil, errors.NewBusiness(errors.CodeParamError, "conversation_id is required")
	}
	if err := s.ensureConversationAccessible(ctx, userID, *req.ConversationId); err != nil {
		return nil, err
	}

	// 参数验证
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// 搜索消息
	var conversationID *string
	if req.ConversationId != nil {
		conversationID = req.ConversationId
	}

	var contentType *string
	if req.ContentType != nil {
		contentType = req.ContentType
	}

	messages, total, err := s.messageRepo.SearchMessages(ctx, req.Keyword, conversationID, contentType, int(req.Limit), int(req.Offset))
	if err != nil {
		logger.Error("Failed to search messages", zap.Error(err))
		return nil, errors.NewBusiness(errors.CodeSearchMessageFailed, "")
	}

	// 转换为protobuf消息
	pbMessages := make([]*messagepb.Message, 0, len(messages))
	for _, msg := range messages {
		pbMsg := s.modelToProto(msg)
		pbMessages = append(pbMessages, pbMsg)
	}

	return &messagepb.SearchMessagesResponse{
		Messages: pbMessages,
		Total:    total,
	}, nil
}

func (s *messageServiceImpl) ensureConversationAccessible(ctx context.Context, userID, conversationID string) error {
	if userID == "" || conversationID == "" {
		return errors.NewBusiness(errors.CodeParamError, "user_id and conversation_id are required")
	}
	if s.conversationClient == nil {
		return errors.NewBusiness(errors.CodeInternalError, "conversation client is not initialized")
	}
	if _, err := s.conversationClient.GetConversation(ctx, &conversationpb.GetConversationRequest{
		UserId:         userID,
		ConversationId: conversationID,
	}); err != nil {
		return errors.NewBusiness(errors.CodeConversationNotFound, "conversation not found")
	}
	return nil
}

// modelToProto 将model转换为protobuf消息
func (s *messageServiceImpl) modelToProto(msg *model.Message) *messagepb.Message {
	pbMsg := &messagepb.Message{
		MessageId:        msg.MessageID,
		ConversationId:   msg.ConversationID,
		ConversationType: msg.ConversationType,
		SenderId:         msg.SenderID,
		ContentType:      msg.ContentType,
		Content:          msg.Content,
		Sequence:         msg.Sequence,
		Status:           int32(msg.Status),
		CreatedAt:        timestamppb.New(msg.CreatedAt),
		UpdatedAt:        timestamppb.New(msg.UpdatedAt),
	}

	if msg.TargetID != "" {
		pbMsg.TargetId = &msg.TargetID
	}

	if msg.ReplyTo != nil {
		pbMsg.ReplyTo = msg.ReplyTo
	}

	if len(msg.AtUsers) > 0 {
		pbMsg.AtUsers = msg.AtUsers
	}

	return pbMsg
}

// publishNewMessageNotification 发布新消息通知
func (s *messageServiceImpl) publishNewMessageNotification(ctx context.Context, msg *model.Message, atUsers []string) error {
	// 解析content获取消息摘要
	contentPreview := s.getContentPreview(msg.Content, msg.ContentType)

	payload := map[string]interface{}{
		"message_id":        msg.MessageID,
		"conversation_id":   msg.ConversationID,
		"conversation_type": msg.ConversationType,
		"target_id":         msg.TargetID,
		"from_user_id":      msg.SenderID,
		"content_type":      msg.ContentType,
		"content":           contentPreview,
		"sent_at":           msg.CreatedAt.Unix(),
		"seq":               msg.Sequence,
	}

	notif := notification.NewNotification(
		notification.TypeMessageNew,
		msg.SenderID,
		notification.PriorityNormal,
	).WithPayload(payload)

	// 根据会话类型决定推送方式
	switch msg.ConversationType {
	case model.ConversationTypeSingle:
		if msg.TargetID == "" {
			logger.Warn("Skip single message notification due to empty target_id",
				zap.String("messageID", msg.MessageID),
				zap.String("conversationID", msg.ConversationID))
			return nil
		}
		return s.notificationPub.PublishToUser(msg.TargetID, notif)
	case model.ConversationTypeGroup:
		groupID := msg.TargetID
		if groupID == "" {
			groupID = msg.ConversationID
		}
		excludedUserIDs := map[string]struct{}{msg.SenderID: {}}
		memberIDs, err := s.listGroupMemberIDs(ctx, msg.SenderID, groupID, excludedUserIDs)
		if err != nil {
			return err
		}
		if len(memberIDs) == 0 {
			return nil
		}
		return s.notificationPub.PublishToUsers(memberIDs, notif)
	}

	return nil
}

func (s *messageServiceImpl) listGroupMemberIDs(ctx context.Context, operatorUserID, groupID string, excludedUserIDs map[string]struct{}) ([]string, error) {
	if s.groupClient == nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "group client is not initialized")
	}

	const pageSizeValue int32 = 100
	pageValue := int32(1)
	pageSize := pageSizeValue
	memberSet := make(map[string]struct{})

	for {
		resp, err := s.groupClient.GetGroupMembers(ctx, &grouppb.GetGroupMembersRequest{
			GroupId:  groupID,
			UserId:   operatorUserID,
			Page:     &pageValue,
			PageSize: &pageSize,
		})
		if err != nil {
			return nil, errors.NewBusiness(errors.CodeInternalError, "failed to load group members")
		}

		if len(resp.Members) == 0 {
			break
		}

		for _, member := range resp.Members {
			if member.UserId == "" {
				continue
			}
			if _, excluded := excludedUserIDs[member.UserId]; excluded {
				continue
			}
			memberSet[member.UserId] = struct{}{}
		}

		if int64(pageValue)*int64(pageSizeValue) >= resp.Total {
			break
		}
		pageValue++
	}

	memberIDs := make([]string, 0, len(memberSet))
	for userID := range memberSet {
		memberIDs = append(memberIDs, userID)
	}
	return memberIDs, nil
}

// publishMentionNotification 发布@提及通知
func (s *messageServiceImpl) publishMentionNotification(msg *model.Message) error {
	if len(msg.AtUsers) == 0 {
		return nil
	}

	contentPreview := s.getContentPreview(msg.Content, msg.ContentType)
	groupID := msg.TargetID
	if groupID == "" {
		groupID = msg.ConversationID
	}

	for _, userID := range msg.AtUsers {
		payload := map[string]interface{}{
			"message_id":   msg.MessageID,
			"group_id":     groupID,
			"from_user_id": msg.SenderID,
			"content":      contentPreview,
			"mention_type": "single",
			"sent_at":      msg.CreatedAt.Unix(),
		}

		notif := notification.NewNotification(
			notification.TypeMessageMentioned,
			msg.SenderID,
			notification.PriorityHigh,
		).WithPayload(payload)

		if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
			logger.Error("Failed to publish mention notification", zap.String("userId", userID), zap.Error(err))
		}
	}

	return nil
}

// publishRecallNotification 发布撤回通知
func (s *messageServiceImpl) publishRecallNotification(ctx context.Context, msg *model.Message, operatorUserID string) error {
	payload := map[string]interface{}{
		"message_id":        msg.MessageID,
		"conversation_id":   msg.ConversationID,
		"conversation_type": msg.ConversationType,
		"target_id":         msg.TargetID,
		"operator_user_id":  operatorUserID,
		"recalled_at":       time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeMessageRecalled,
		operatorUserID,
		notification.PriorityNormal,
	).WithPayload(payload)

	switch msg.ConversationType {
	case model.ConversationTypeSingle:
		receiverSet := map[string]struct{}{}
		if msg.SenderID != "" {
			receiverSet[msg.SenderID] = struct{}{}
		}
		if msg.TargetID != "" {
			receiverSet[msg.TargetID] = struct{}{}
		}
		if len(receiverSet) == 0 {
			logger.Warn("Skip recall notification due to empty receivers",
				zap.String("messageID", msg.MessageID),
				zap.String("conversationID", msg.ConversationID))
			return nil
		}

		receiverIDs := make([]string, 0, len(receiverSet))
		for userID := range receiverSet {
			receiverIDs = append(receiverIDs, userID)
		}
		return s.notificationPub.PublishToUsers(receiverIDs, notif)

	case model.ConversationTypeGroup:
		groupID := msg.TargetID
		if groupID == "" {
			groupID = msg.ConversationID
		}
		memberIDs, err := s.listGroupMemberIDs(ctx, operatorUserID, groupID, nil)
		if err != nil {
			return err
		}
		if len(memberIDs) == 0 {
			return nil
		}
		return s.notificationPub.PublishToUsers(memberIDs, notif)
	}

	return nil
}

// publishReadReceiptNotification 发布已读回执通知
func (s *messageServiceImpl) publishReadReceiptNotification(receipt *model.MessageReadReceipt) error {
	payload := map[string]interface{}{
		"conversation_id":   receipt.ConversationID,
		"conversation_type": receipt.ConversationType,
		"target_id":         receipt.TargetID,
		"reader_user_id":    receipt.UserID,
		"last_read_seq":     receipt.LastReadSeq,
		"read_at":           receipt.ReadAt.Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeMessageReadReceipt,
		receipt.UserID,
		notification.PriorityLow,
	).WithPayload(payload)

	// 单聊已读回执：通知对方用户
	if receipt.ConversationType == model.ConversationTypeSingle {
		if receipt.TargetID == "" {
			logger.Warn("Skip read receipt notification due to empty target_id",
				zap.String("conversationID", receipt.ConversationID),
				zap.String("readerUserID", receipt.UserID))
			return nil
		}
		return s.notificationPub.PublishToUser(receipt.TargetID, notif)
	}

	return nil
}

// getContentPreview 获取内容预览
func (s *messageServiceImpl) getContentPreview(content, contentType string) string {
	switch contentType {
	case model.ContentTypeText:
		// 解析JSON获取文本内容
		var textContent struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(content), &textContent); err == nil {
			if len(textContent.Text) > 100 {
				return textContent.Text[:100] + "..."
			}
			return textContent.Text
		}
		return "[文本消息]"
	case model.ContentTypeImage:
		return "[图片]"
	case model.ContentTypeVideo:
		return "[视频]"
	case model.ContentTypeAudio:
		return "[语音]"
	case model.ContentTypeFile:
		return "[文件]"
	case model.ContentTypeLocation:
		return "[位置]"
	case model.ContentTypeCard:
		return "[名片]"
	default:
		return "[消息]"
	}
}
