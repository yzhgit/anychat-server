package repository

import (
	"context"

	"github.com/anychat/server/internal/message/model"
	"gorm.io/gorm"
)

// ReadReceiptRepository 已读回执仓库接口
type ReadReceiptRepository interface {
	Upsert(ctx context.Context, receipt *model.MessageReadReceipt) error
	GetByConversationAndUser(ctx context.Context, conversationID, userID string) (*model.MessageReadReceipt, error)
	GetByConversation(ctx context.Context, conversationID string) ([]*model.MessageReadReceipt, error)
	GetByUser(ctx context.Context, userID string) ([]*model.MessageReadReceipt, error)
	Delete(ctx context.Context, conversationID, userID string) error
	WithTx(tx *gorm.DB) ReadReceiptRepository
}

// readReceiptRepositoryImpl 已读回执仓库实现
type readReceiptRepositoryImpl struct {
	db *gorm.DB
}

// NewReadReceiptRepository 创建已读回执仓库
func NewReadReceiptRepository(db *gorm.DB) ReadReceiptRepository {
	return &readReceiptRepositoryImpl{db: db}
}

// Upsert 创建或更新已读回执
func (r *readReceiptRepositoryImpl) Upsert(ctx context.Context, receipt *model.MessageReadReceipt) error {
	// 使用ON CONFLICT更新
	return r.db.WithContext(ctx).
		Clauses(
			// PostgreSQL upsert语法
			gorm.Expr("ON CONFLICT (conversation_id, user_id) DO UPDATE SET last_read_seq = ?, last_read_message_id = ?, read_at = ?",
				receipt.LastReadSeq, receipt.LastReadMessageID, receipt.ReadAt),
		).
		Create(receipt).Error
}

// GetByConversationAndUser 根据会话和用户获取已读回执
func (r *readReceiptRepositoryImpl) GetByConversationAndUser(ctx context.Context, conversationID, userID string) (*model.MessageReadReceipt, error) {
	var receipt model.MessageReadReceipt
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&receipt).Error
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

// GetByConversation 获取会话的所有已读回执
func (r *readReceiptRepositoryImpl) GetByConversation(ctx context.Context, conversationID string) ([]*model.MessageReadReceipt, error) {
	var receipts []*model.MessageReadReceipt
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("read_at DESC").
		Find(&receipts).Error
	return receipts, err
}

// GetByUser 获取用户的所有已读回执
func (r *readReceiptRepositoryImpl) GetByUser(ctx context.Context, userID string) ([]*model.MessageReadReceipt, error) {
	var receipts []*model.MessageReadReceipt
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("read_at DESC").
		Find(&receipts).Error
	return receipts, err
}

// Delete 删除已读回执
func (r *readReceiptRepositoryImpl) Delete(ctx context.Context, conversationID, userID string) error {
	return r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&model.MessageReadReceipt{}).Error
}

// WithTx 使用事务
func (r *readReceiptRepositoryImpl) WithTx(tx *gorm.DB) ReadReceiptRepository {
	return &readReceiptRepositoryImpl{db: tx}
}
