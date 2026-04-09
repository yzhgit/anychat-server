package repository

import (
	"context"

	"github.com/anychat/server/internal/friend/model"
	"gorm.io/gorm"
)

// BlacklistRepository is the blacklist repository interface
type BlacklistRepository interface {
	Create(ctx context.Context, blacklist *model.Blacklist) error
	GetByUserAndBlocked(ctx context.Context, userID, blockedUserID string) (*model.Blacklist, error)
	GetBlacklist(ctx context.Context, userID string) ([]*model.Blacklist, error)
	Delete(ctx context.Context, userID, blockedUserID string) error
	IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error)
	WithTx(tx *gorm.DB) BlacklistRepository
}

// blacklistRepositoryImpl is the blacklist repository implementation
type blacklistRepositoryImpl struct {
	db *gorm.DB
}

// NewBlacklistRepository creates a new blacklist repository
func NewBlacklistRepository(db *gorm.DB) BlacklistRepository {
	return &blacklistRepositoryImpl{db: db}
}

// Create adds to blacklist
func (r *blacklistRepositoryImpl) Create(ctx context.Context, blacklist *model.Blacklist) error {
	return r.db.WithContext(ctx).Create(blacklist).Error
}

// GetByUserAndBlocked retrieves a blacklist record
func (r *blacklistRepositoryImpl) GetByUserAndBlocked(ctx context.Context, userID, blockedUserID string) (*model.Blacklist, error) {
	var blacklist model.Blacklist
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).
		First(&blacklist).Error
	if err != nil {
		return nil, err
	}
	return &blacklist, nil
}

// GetBlacklist retrieves the blacklist
func (r *blacklistRepositoryImpl) GetBlacklist(ctx context.Context, userID string) ([]*model.Blacklist, error) {
	var blacklist []*model.Blacklist
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&blacklist).Error
	return blacklist, err
}

// Delete removes from blacklist
func (r *blacklistRepositoryImpl) Delete(ctx context.Context, userID, blockedUserID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).
		Delete(&model.Blacklist{}).Error
}

// IsBlocked checks if user is blocked (bidirectional check: A blocks B or B blocks A)
func (r *blacklistRepositoryImpl) IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Blacklist{}).
		Where("(user_id = ? AND blocked_user_id = ?) OR (user_id = ? AND blocked_user_id = ?)",
			userID, targetUserID, targetUserID, userID).
		Count(&count).Error
	return count > 0, err
}

// WithTx uses transaction
func (r *blacklistRepositoryImpl) WithTx(tx *gorm.DB) BlacklistRepository {
	return &blacklistRepositoryImpl{db: tx}
}
