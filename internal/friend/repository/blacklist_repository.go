package repository

import (
	"context"

	"github.com/anychat/server/internal/friend/model"
	"gorm.io/gorm"
)

// BlacklistRepository 黑名单仓库接口
type BlacklistRepository interface {
	Create(ctx context.Context, blacklist *model.Blacklist) error
	GetByUserAndBlocked(ctx context.Context, userID, blockedUserID string) (*model.Blacklist, error)
	GetBlacklist(ctx context.Context, userID string) ([]*model.Blacklist, error)
	Delete(ctx context.Context, userID, blockedUserID string) error
	IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error)
	WithTx(tx *gorm.DB) BlacklistRepository
}

// blacklistRepositoryImpl 黑名单仓库实现
type blacklistRepositoryImpl struct {
	db *gorm.DB
}

// NewBlacklistRepository 创建黑名单仓库
func NewBlacklistRepository(db *gorm.DB) BlacklistRepository {
	return &blacklistRepositoryImpl{db: db}
}

// Create 添加黑名单
func (r *blacklistRepositoryImpl) Create(ctx context.Context, blacklist *model.Blacklist) error {
	return r.db.WithContext(ctx).Create(blacklist).Error
}

// GetByUserAndBlocked 获取黑名单记录
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

// GetBlacklist 获取黑名单列表
func (r *blacklistRepositoryImpl) GetBlacklist(ctx context.Context, userID string) ([]*model.Blacklist, error) {
	var blacklist []*model.Blacklist
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&blacklist).Error
	return blacklist, err
}

// Delete 删除黑名单
func (r *blacklistRepositoryImpl) Delete(ctx context.Context, userID, blockedUserID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).
		Delete(&model.Blacklist{}).Error
}

// IsBlocked 检查是否被拉黑（双向检查：A拉黑B 或 B拉黑A）
func (r *blacklistRepositoryImpl) IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Blacklist{}).
		Where("(user_id = ? AND blocked_user_id = ?) OR (user_id = ? AND blocked_user_id = ?)",
			userID, targetUserID, targetUserID, userID).
		Count(&count).Error
	return count > 0, err
}

// WithTx 使用事务
func (r *blacklistRepositoryImpl) WithTx(tx *gorm.DB) BlacklistRepository {
	return &blacklistRepositoryImpl{db: tx}
}
