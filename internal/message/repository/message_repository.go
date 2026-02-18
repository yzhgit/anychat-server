package repository

import (
	"context"

	"github.com/anychat/server/internal/message/model"
	"gorm.io/gorm"
)

// MessageRepository 消息仓库接口
type MessageRepository interface {
	Create(ctx context.Context, message *model.Message) error
	CreateBatch(ctx context.Context, messages []*model.Message) error
	GetByID(ctx context.Context, id int64) (*model.Message, error)
	GetByMessageID(ctx context.Context, messageID string) (*model.Message, error)
	GetByConversation(ctx context.Context, conversationID string, startSeq, endSeq int64, limit int, reverse bool) ([]*model.Message, error)
	GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*model.Message, error)
	GetBySender(ctx context.Context, senderID string, limit, offset int) ([]*model.Message, error)
	UpdateStatus(ctx context.Context, messageID string, status int16) error
	Delete(ctx context.Context, messageID string) error
	CountByConversation(ctx context.Context, conversationID string) (int64, error)
	CountUnreadByConversation(ctx context.Context, conversationID string, lastReadSeq int64) (int64, error)
	SearchMessages(ctx context.Context, keyword string, conversationID *string, contentType *string, limit, offset int) ([]*model.Message, int64, error)
	GetByReplyTo(ctx context.Context, replyToMessageID string) ([]*model.Message, error)
	WithTx(tx *gorm.DB) MessageRepository
}

// messageRepositoryImpl 消息仓库实现
type messageRepositoryImpl struct {
	db *gorm.DB
}

// NewMessageRepository 创建消息仓库
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepositoryImpl{db: db}
}

// Create 创建消息
func (r *messageRepositoryImpl) Create(ctx context.Context, message *model.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

// CreateBatch 批量创建消息
func (r *messageRepositoryImpl) CreateBatch(ctx context.Context, messages []*model.Message) error {
	return r.db.WithContext(ctx).Create(messages).Error
}

// GetByID 根据ID获取消息
func (r *messageRepositoryImpl) GetByID(ctx context.Context, id int64) (*model.Message, error) {
	var message model.Message
	err := r.db.WithContext(ctx).
		Where("id = ? AND status != ?", id, model.MessageStatusDeleted).
		First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// GetByMessageID 根据消息ID获取消息
func (r *messageRepositoryImpl) GetByMessageID(ctx context.Context, messageID string) (*model.Message, error) {
	var message model.Message
	err := r.db.WithContext(ctx).
		Where("message_id = ? AND status != ?", messageID, model.MessageStatusDeleted).
		First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// GetByConversation 获取会话消息列表（支持序列号范围查询）
func (r *messageRepositoryImpl) GetByConversation(ctx context.Context, conversationID string, startSeq, endSeq int64, limit int, reverse bool) ([]*model.Message, error) {
	var messages []*model.Message
	query := r.db.WithContext(ctx).
		Where("conversation_id = ? AND status = ?", conversationID, model.MessageStatusNormal)

	if startSeq > 0 {
		query = query.Where("sequence >= ?", startSeq)
	}
	if endSeq > 0 {
		query = query.Where("sequence <= ?", endSeq)
	}

	if reverse {
		query = query.Order("sequence DESC")
	} else {
		query = query.Order("sequence ASC")
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&messages).Error
	return messages, err
}

// GetLatestByConversation 获取会话最新消息
func (r *messageRepositoryImpl) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*model.Message, error) {
	var messages []*model.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND status = ?", conversationID, model.MessageStatusNormal).
		Order("sequence DESC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

// GetBySender 根据发送者获取消息列表
func (r *messageRepositoryImpl) GetBySender(ctx context.Context, senderID string, limit, offset int) ([]*model.Message, error) {
	var messages []*model.Message
	err := r.db.WithContext(ctx).
		Where("sender_id = ? AND status = ?", senderID, model.MessageStatusNormal).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

// UpdateStatus 更新消息状态
func (r *messageRepositoryImpl) UpdateStatus(ctx context.Context, messageID string, status int16) error {
	return r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("message_id = ?", messageID).
		Update("status", status).Error
}

// Delete 删除消息（软删除）
func (r *messageRepositoryImpl) Delete(ctx context.Context, messageID string) error {
	return r.UpdateStatus(ctx, messageID, model.MessageStatusDeleted)
}

// CountByConversation 统计会话消息数量
func (r *messageRepositoryImpl) CountByConversation(ctx context.Context, conversationID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ? AND status = ?", conversationID, model.MessageStatusNormal).
		Count(&count).Error
	return count, err
}

// CountUnreadByConversation 统计会话未读消息数量
func (r *messageRepositoryImpl) CountUnreadByConversation(ctx context.Context, conversationID string, lastReadSeq int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ? AND sequence > ? AND status = ?", conversationID, lastReadSeq, model.MessageStatusNormal).
		Count(&count).Error
	return count, err
}

// SearchMessages 搜索消息
func (r *messageRepositoryImpl) SearchMessages(ctx context.Context, keyword string, conversationID *string, contentType *string, limit, offset int) ([]*model.Message, int64, error) {
	var messages []*model.Message
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("status = ?", model.MessageStatusNormal)

	// 关键词搜索（使用JSONB包含查询）
	if keyword != "" {
		query = query.Where("content::text ILIKE ?", "%"+keyword+"%")
	}

	// 会话过滤
	if conversationID != nil && *conversationID != "" {
		query = query.Where("conversation_id = ?", *conversationID)
	}

	// 内容类型过滤
	if contentType != nil && *contentType != "" {
		query = query.Where("content_type = ?", *contentType)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error

	return messages, total, err
}

// GetByReplyTo 获取回复某条消息的所有消息
func (r *messageRepositoryImpl) GetByReplyTo(ctx context.Context, replyToMessageID string) ([]*model.Message, error) {
	var messages []*model.Message
	err := r.db.WithContext(ctx).
		Where("reply_to = ? AND status = ?", replyToMessageID, model.MessageStatusNormal).
		Order("created_at ASC").
		Find(&messages).Error
	return messages, err
}

// WithTx 使用事务
func (r *messageRepositoryImpl) WithTx(tx *gorm.DB) MessageRepository {
	return &messageRepositoryImpl{db: tx}
}
