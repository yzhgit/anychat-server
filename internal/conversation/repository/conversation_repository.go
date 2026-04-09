package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/conversation/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ConversationRepository is the conversation repository interface
type ConversationRepository interface {
	// Upsert creates or updates a conversation
	Upsert(ctx context.Context, conversation *model.Conversation) error
	// GetByID retrieves a conversation by conversation ID
	GetByID(ctx context.Context, conversationID string) (*model.Conversation, error)
	// GetByUserAndTarget retrieves a conversation by user ID and target ID
	GetByUserAndTarget(ctx context.Context, userID, conversationType, targetID string) (*model.Conversation, error)
	// ListByUser retrieves the user's conversation list
	ListByUser(ctx context.Context, userID string, limit int, updatedBefore *time.Time) ([]*model.Conversation, error)
	// Delete deletes a conversation
	Delete(ctx context.Context, userID, conversationID string) error
	// SetPinned sets pinned status
	SetPinned(ctx context.Context, userID, conversationID string, pinned bool, pinTime *time.Time) error
	// SetMuted sets muted status
	SetMuted(ctx context.Context, userID, conversationID string, muted bool) error
	// SetBurnAfterReading sets burn after reading duration
	SetBurnAfterReading(ctx context.Context, userID, conversationID string, duration int32) error
	// SetAutoDelete sets auto delete duration
	SetAutoDelete(ctx context.Context, userID, conversationID string, duration int32) error
	// ClearUnread clears unread count
	ClearUnread(ctx context.Context, userID, conversationID string) error
	// IncrUnread increments unread count
	IncrUnread(ctx context.Context, userID, conversationID string, count int32) error
	// SumUnread counts user's total unread count
	SumUnread(ctx context.Context, userID string) (int32, error)
	// WithTx uses transaction
	WithTx(tx *gorm.DB) ConversationRepository
}

// conversationRepositoryImpl is the conversation repository implementation
type conversationRepositoryImpl struct {
	db *gorm.DB
}

// NewConversationRepository creates a new conversation repository
func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return &conversationRepositoryImpl{db: db}
}

// Upsert creates or updates a conversation (updates last message info on conflict)
func (r *conversationRepositoryImpl) Upsert(ctx context.Context, conversation *model.Conversation) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "conversation_type"}, {Name: "target_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"last_message_id",
				"last_message_content",
				"last_message_time",
				"updated_at",
			}),
		}).
		Create(conversation).Error
}

// GetByID retrieves a conversation by conversation ID
func (r *conversationRepositoryImpl) GetByID(ctx context.Context, conversationID string) (*model.Conversation, error) {
	var conversation model.Conversation
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		First(&conversation).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

// GetByUserAndTarget retrieves a conversation by user ID and target ID
func (r *conversationRepositoryImpl) GetByUserAndTarget(ctx context.Context, userID, conversationType, targetID string) (*model.Conversation, error) {
	var conversation model.Conversation
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND conversation_type = ? AND target_id = ?", userID, conversationType, targetID).
		First(&conversation).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

// ListByUser retrieves user conversation list (sorted by pinned + last message time)
func (r *conversationRepositoryImpl) ListByUser(ctx context.Context, userID string, limit int, updatedBefore *time.Time) ([]*model.Conversation, error) {
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if updatedBefore != nil {
		q = q.Where("updated_at < ?", updatedBefore)
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var conversations []*model.Conversation
	err := q.Order("is_pinned DESC, COALESCE(last_message_time, created_at) DESC").
		Limit(limit).
		Find(&conversations).Error
	return conversations, err
}

// Delete deletes a conversation (only deletes conversation belonging to the user)
func (r *conversationRepositoryImpl) Delete(ctx context.Context, userID, conversationID string) error {
	return r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&model.Conversation{}).Error
}

// SetPinned sets pinned status
func (r *conversationRepositoryImpl) SetPinned(ctx context.Context, userID, conversationID string, pinned bool, pinTime *time.Time) error {
	updates := map[string]interface{}{
		"is_pinned": pinned,
		"pin_time":  pinTime,
	}
	return r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(updates).Error
}

// SetMuted sets muted status
func (r *conversationRepositoryImpl) SetMuted(ctx context.Context, userID, conversationID string, muted bool) error {
	return r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("is_muted", muted).Error
}

// SetBurnAfterReading sets burn after reading duration
func (r *conversationRepositoryImpl) SetBurnAfterReading(ctx context.Context, userID, conversationID string, duration int32) error {
	return r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("burn_after_reading", duration).Error
}

// SetAutoDelete sets auto delete duration
func (r *conversationRepositoryImpl) SetAutoDelete(ctx context.Context, userID, conversationID string, duration int32) error {
	return r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("auto_delete_duration", duration).Error
}

// ClearUnread clears unread count
func (r *conversationRepositoryImpl) ClearUnread(ctx context.Context, userID, conversationID string) error {
	return r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("unread_count", 0).Error
}

// IncrUnread increments unread count
func (r *conversationRepositoryImpl) IncrUnread(ctx context.Context, userID, conversationID string, count int32) error {
	return r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + ?", count)).Error
}

// SumUnread counts all unread counts for user (muted conversations are not included)
func (r *conversationRepositoryImpl) SumUnread(ctx context.Context, userID string) (int32, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.Conversation{}).
		Where("user_id = ? AND is_muted = false", userID).
		Select("COALESCE(SUM(unread_count), 0)").
		Scan(&total).Error
	return int32(total), err
}

// WithTx returns a repository instance using transaction
func (r *conversationRepositoryImpl) WithTx(tx *gorm.DB) ConversationRepository {
	return &conversationRepositoryImpl{db: tx}
}
