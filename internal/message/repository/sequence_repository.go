package repository

import (
	"context"

	"github.com/anychat/server/internal/message/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SequenceRepository 序列号仓库接口
type SequenceRepository interface {
	GetOrCreate(ctx context.Context, conversationID string) (*model.ConversationSequence, error)
	IncrementAndGet(ctx context.Context, conversationID string) (int64, error)
	GetCurrentSeq(ctx context.Context, conversationID string) (int64, error)
	Reset(ctx context.Context, conversationID string) error
	Delete(ctx context.Context, conversationID string) error
	WithTx(tx *gorm.DB) SequenceRepository
}

// sequenceRepositoryImpl 序列号仓库实现
type sequenceRepositoryImpl struct {
	db *gorm.DB
}

// NewSequenceRepository 创建序列号仓库
func NewSequenceRepository(db *gorm.DB) SequenceRepository {
	return &sequenceRepositoryImpl{db: db}
}

// GetOrCreate 获取或创建序列号记录
func (r *sequenceRepositoryImpl) GetOrCreate(ctx context.Context, conversationID string) (*model.ConversationSequence, error) {
	var seq model.ConversationSequence

	// 尝试获取
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		First(&seq).Error

	if err == nil {
		return &seq, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 不存在则创建
	seq = model.ConversationSequence{
		ConversationID: conversationID,
		CurrentSeq:     0,
	}

	err = r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&seq).Error

	if err != nil {
		// 如果并发创建失败，再次获取
		err = r.db.WithContext(ctx).
			Where("conversation_id = ?", conversationID).
			First(&seq).Error
	}

	return &seq, err
}

// IncrementAndGet 递增并获取序列号（原子操作）
func (r *sequenceRepositoryImpl) IncrementAndGet(ctx context.Context, conversationID string) (int64, error) {
	// 确保记录存在
	if _, err := r.GetOrCreate(ctx, conversationID); err != nil {
		return 0, err
	}

	// 原子递增
	result := r.db.WithContext(ctx).
		Model(&model.ConversationSequence{}).
		Where("conversation_id = ?", conversationID).
		UpdateColumn("current_seq", gorm.Expr("current_seq + 1"))

	if result.Error != nil {
		return 0, result.Error
	}

	// 获取更新后的值
	var seq model.ConversationSequence
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		First(&seq).Error

	return seq.CurrentSeq, err
}

// GetCurrentSeq 获取当前序列号
func (r *sequenceRepositoryImpl) GetCurrentSeq(ctx context.Context, conversationID string) (int64, error) {
	var seq model.ConversationSequence
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		First(&seq).Error

	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	return seq.CurrentSeq, nil
}

// Reset 重置序列号
func (r *sequenceRepositoryImpl) Reset(ctx context.Context, conversationID string) error {
	return r.db.WithContext(ctx).
		Model(&model.ConversationSequence{}).
		Where("conversation_id = ?", conversationID).
		Update("current_seq", 0).Error
}

// Delete 删除序列号记录
func (r *sequenceRepositoryImpl) Delete(ctx context.Context, conversationID string) error {
	return r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Delete(&model.ConversationSequence{}).Error
}

// WithTx 使用事务
func (r *sequenceRepositoryImpl) WithTx(tx *gorm.DB) SequenceRepository {
	return &sequenceRepositoryImpl{db: tx}
}
