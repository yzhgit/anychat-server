package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/group/model"
	"gorm.io/gorm"
)

// GroupSettingRepository defines the group settings repository interface
type GroupSettingRepository interface {
	Create(ctx context.Context, setting *model.GroupSetting) error
	GetSettings(ctx context.Context, groupID string) (*model.GroupSetting, error)
	UpdateSettings(ctx context.Context, groupID string, updates map[string]interface{}) error
	Delete(ctx context.Context, groupID string) error
	WithTx(tx *gorm.DB) GroupSettingRepository
}

// groupSettingRepositoryImpl is the group settings repository implementation
type groupSettingRepositoryImpl struct {
	db *gorm.DB
}

// NewGroupSettingRepository creates a new group settings repository
func NewGroupSettingRepository(db *gorm.DB) GroupSettingRepository {
	return &groupSettingRepositoryImpl{db: db}
}

// Create creates group settings
func (r *groupSettingRepositoryImpl) Create(ctx context.Context, setting *model.GroupSetting) error {
	return r.db.WithContext(ctx).Create(setting).Error
}

// GetSettings gets group settings
func (r *groupSettingRepositoryImpl) GetSettings(ctx context.Context, groupID string) (*model.GroupSetting, error) {
	var setting model.GroupSetting
	err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

// UpdateSettings updates group settings
func (r *groupSettingRepositoryImpl) UpdateSettings(ctx context.Context, groupID string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return r.db.WithContext(ctx).
		Model(&model.GroupSetting{}).
		Where("group_id = ?", groupID).
		Updates(updates).Error
}

// Delete deletes group settings
func (r *groupSettingRepositoryImpl) Delete(ctx context.Context, groupID string) error {
	return r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Delete(&model.GroupSetting{}).Error
}

// WithTx uses transaction
func (r *groupSettingRepositoryImpl) WithTx(tx *gorm.DB) GroupSettingRepository {
	return &groupSettingRepositoryImpl{db: tx}
}
