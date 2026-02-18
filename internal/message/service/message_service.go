package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/anychat/server/internal/message/model"
	"github.com/anychat/server/internal/message/repository"
	messagepb "github.com/anychat/server/api/proto/message"
	"github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MessageService 消息服务接口
type MessageService interface {
	SendMessage(ctx context.Context, req *messagepb.SendMessageRequest) (*messagepb.SendMessageResponse, error)
	GetMessages(ctx context.Context, req *messagepb.GetMessagesRequest) (*messagepb.GetMessagesResponse, error)
	GetMessageById(ctx context.Context, messageID string) (*messagepb.Message, error)
	RecallMessage(ctx context.Context, messageID, userID string) error
	DeleteMessage(ctx context.Context, messageID, userID string) error
	MarkAsRead(ctx context.Context, req *messagepb.MarkAsReadRequest) error
	GetUnreadCount(ctx context.Context, conversationID, userID string, lastReadSeq *int64) (*messagepb.GetUnreadCountResponse, error)
	GetReadReceipts(ctx context.Context, conversationID, userID string) (*messagepb.GetReadReceiptsResponse, error)
	GetConversationSequence(ctx context.Context, conversationID string) (int64, error)
	SearchMessages(ctx context.Context, req *messagepb.SearchMessagesRequest) (*messagepb.SearchMessagesResponse, error)
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

// messageServiceImpl 消息服务实现
type messageServiceImpl struct {
	messageRepo      MessageRepo
	readReceiptRepo  ReadReceiptRepo
	sequenceRepo     SequenceRepo
	notificationPub  notification.Publisher
	db               *gorm.DB
}

// NewMessageService 创建消息服务
func NewMessageService(
	messageRepo repository.MessageRepository,
	readReceiptRepo repository.ReadReceiptRepository,
	sequenceRepo repository.SequenceRepository,
	notificationPub notification.Publisher,
	db *gorm.DB,
) MessageService {
	return &messageServiceImpl{
		messageRepo:     messageRepo,
		readReceiptRepo: readReceiptRepo,
		sequenceRepo:    sequenceRepo,
		notificationPub: notificationPub,
		db:              db,
	}
}

// SendMessage 发送消息
func (s *messageServiceImpl) SendMessage(ctx context.Context, req *messagepb.SendMessageRequest) (*messagepb.SendMessageResponse, error) {
	// 1. 生成消息ID
	messageID := uuid.New().String()

	// 2. 获取下一个序列号（原子操作）
	sequence, err := s.sequenceRepo.IncrementAndGet(ctx, req.ConversationId)
	if err != nil {
		logger.Error("Failed to increment sequence", zap.Error(err))
		return nil, errors.NewBusiness(errors.CodeSequenceGenerateFailed, "")
	}

	// 3. 创建消息记录
	now := time.Now()
	message := &model.Message{
		MessageID:        messageID,
		ConversationID:   req.ConversationId,
		ConversationType: req.ConversationType,
		SenderID:         req.SenderId,
		ContentType:      req.ContentType,
		Content:          req.Content,
		Sequence:         sequence,
		Status:           model.MessageStatusNormal,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// 设置可选字段
	if req.ReplyTo != nil {
		message.ReplyTo = req.ReplyTo
	}
	if len(req.AtUsers) > 0 {
		message.AtUsers = req.AtUsers
	}

	// 4. 保存消息到数据库
	if err := s.messageRepo.Create(ctx, message); err != nil {
		logger.Error("Failed to create message", zap.Error(err))
		return nil, errors.NewBusiness(errors.CodeMessageSendFailed, "")
	}

	// 5. 发布新消息通知
	if err := s.publishNewMessageNotification(message, req.AtUsers); err != nil {
		logger.Error("Failed to publish message notification", zap.Error(err))
		// 通知失败不影响消息发送
	}

	// 6. 如果有@提及，发布@通知
	if len(req.AtUsers) > 0 {
		if err := s.publishMentionNotification(message); err != nil {
			logger.Error("Failed to publish mention notification", zap.Error(err))
		}
	}

	return &messagepb.SendMessageResponse{
		MessageId: messageID,
		Sequence:  sequence,
		Timestamp: timestamppb.New(now),
	}, nil
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

	// 获取消息列表
	startSeq := int64(0)
	endSeq := int64(0)
	if req.StartSeq != nil {
		startSeq = *req.StartSeq
	}
	if req.EndSeq != nil {
		endSeq = *req.EndSeq
	}

	messages, err := s.messageRepo.GetByConversation(ctx, req.ConversationId, startSeq, endSeq, int(req.Limit), req.Reverse)
	if err != nil {
		logger.Error("Failed to get messages", zap.Error(err))
		return nil, errors.NewBusiness(errors.CodeInternalError, "Failed to retrieve messages")
	}

	// 转换为protobuf消息
	pbMessages := make([]*messagepb.Message, 0, len(messages))
	for _, msg := range messages {
		pbMsg := s.modelToProto(msg)
		pbMessages = append(pbMessages, pbMsg)
	}

	// 检查是否还有更多消息
	hasMore := len(messages) == int(req.Limit)

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
	if err := s.publishRecallNotification(message, userID); err != nil {
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
func (s *messageServiceImpl) MarkAsRead(ctx context.Context, req *messagepb.MarkAsReadRequest) error {
	// 创建或更新已读回执
	receipt := &model.MessageReadReceipt{
		ConversationID:   req.ConversationId,
		ConversationType: req.ConversationType,
		UserID:           req.UserId,
		LastReadSeq:      req.LastReadSeq,
		ReadAt:           time.Now(),
	}

	if req.LastReadMessageId != nil {
		receipt.LastReadMessageID = req.LastReadMessageId
	}

	if err := s.readReceiptRepo.Upsert(ctx, receipt); err != nil {
		logger.Error("Failed to upsert read receipt", zap.Error(err))
		return errors.NewBusiness(errors.CodeMarkReadFailed, "")
	}

	// 发布已读回执通知（单聊时通知对方）
	if req.ConversationType == model.ConversationTypeSingle {
		if err := s.publishReadReceiptNotification(receipt); err != nil {
			logger.Error("Failed to publish read receipt notification", zap.Error(err))
		}
	}

	return nil
}

// GetUnreadCount 获取未读消息数
func (s *messageServiceImpl) GetUnreadCount(ctx context.Context, conversationID, userID string, lastReadSeq *int64) (*messagepb.GetUnreadCountResponse, error) {
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
func (s *messageServiceImpl) SearchMessages(ctx context.Context, req *messagepb.SearchMessagesRequest) (*messagepb.SearchMessagesResponse, error) {
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

	if msg.ReplyTo != nil {
		pbMsg.ReplyTo = msg.ReplyTo
	}

	if len(msg.AtUsers) > 0 {
		pbMsg.AtUsers = msg.AtUsers
	}

	return pbMsg
}

// publishNewMessageNotification 发布新消息通知
func (s *messageServiceImpl) publishNewMessageNotification(msg *model.Message, atUsers []string) error {
	// 解析content获取消息摘要
	contentPreview := s.getContentPreview(msg.Content, msg.ContentType)

	payload := map[string]interface{}{
		"message_id":      msg.MessageID,
		"conversation_id": msg.ConversationID,
		"conversation_type": msg.ConversationType,
		"from_user_id":    msg.SenderID,
		"content_type":    msg.ContentType,
		"content":         contentPreview,
		"sent_at":         msg.CreatedAt.Unix(),
		"seq":             msg.Sequence,
	}

	notif := notification.NewNotification(
		notification.TypeMessageNew,
		msg.SenderID,
		notification.PriorityNormal,
	).WithPayload(payload)

	// 根据会话类型决定推送方式
	if msg.ConversationType == model.ConversationTypeSingle {
		// 单聊：从conversation_id中提取接收者ID
		// conversation_id格式: conv-{userId1}-{userId2}
		// TODO: 这里需要根据实际的conversation_id格式来解析
		_ = notif // 暂时忽略，等待conversation_id解析逻辑
		return nil // 暂时跳过
	} else if msg.ConversationType == model.ConversationTypeGroup {
		// 群聊：发布到群组主题
		return s.notificationPub.PublishToGroup(msg.ConversationID, notif)
	}

	return nil
}

// publishMentionNotification 发布@提及通知
func (s *messageServiceImpl) publishMentionNotification(msg *model.Message) error {
	if len(msg.AtUsers) == 0 {
		return nil
	}

	contentPreview := s.getContentPreview(msg.Content, msg.ContentType)

	for _, userID := range msg.AtUsers {
		payload := map[string]interface{}{
			"message_id":   msg.MessageID,
			"group_id":     msg.ConversationID,
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
func (s *messageServiceImpl) publishRecallNotification(msg *model.Message, operatorUserID string) error {
	payload := map[string]interface{}{
		"message_id":        msg.MessageID,
		"conversation_id":   msg.ConversationID,
		"conversation_type": msg.ConversationType,
		"operator_user_id":  operatorUserID,
		"recalled_at":       time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeMessageRecalled,
		operatorUserID,
		notification.PriorityNormal,
	).WithPayload(payload)

	// 根据会话类型推送
	if msg.ConversationType == model.ConversationTypeGroup {
		return s.notificationPub.PublishToGroup(msg.ConversationID, notif)
	}

	return nil
}

// publishReadReceiptNotification 发布已读回执通知
func (s *messageServiceImpl) publishReadReceiptNotification(receipt *model.MessageReadReceipt) error {
	payload := map[string]interface{}{
		"conversation_id":   receipt.ConversationID,
		"conversation_type": receipt.ConversationType,
		"reader_user_id":    receipt.UserID,
		"last_read_seq":     receipt.LastReadSeq,
		"read_at":           receipt.ReadAt.Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeMessageReadReceipt,
		receipt.UserID,
		notification.PriorityLow,
	).WithPayload(payload)

	// 单聊已读回执：需要通知对方
	// TODO: 从conversation_id中提取对方用户ID
	_ = notif // 暂时忽略，等待conversation_id解析逻辑
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
