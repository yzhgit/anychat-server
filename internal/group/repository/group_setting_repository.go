package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/group/model"
	"gorm.io/gorm"
)

// GroupSettingRepository 群组设置仓库接口
type GroupSettingRepository interface {
	Create(ctx context.Context, setting *model.GroupSetting) error
	GetSettings(ctx context.Context, groupID string) (*model.GroupSetting, error)
	UpdateSettings(ctx context.Context, groupID string, updates map[string]interface{}) error
	Delete(ctx context.Context, groupID string) error
	WithTx(tx *gorm.DB) GroupSettingRepository
}

// groupSettingRepositoryImpl 群组设置仓库实现
type groupSettingRepositoryImpl struct {
	db *gorm.DB
}

// NewGroupSettingRepository 创建群组设置仓库
func NewGroupSettingRepository(db *gorm.DB) GroupSettingRepository {
	return &groupSettingRepositoryImpl{db: db}
}

// Create 创建群组设置
func (r *groupSettingRepositoryImpl) Create(ctx context.Context, setting *model.GroupSetting) error {
	return r.db.WithContext(ctx).Create(setting).Error
}

// GetSettings 获取群组设置
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

// UpdateSettings 更新群组设置
func (r *groupSettingRepositoryImpl) UpdateSettings(ctx context.Context, groupID string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return r.db.WithContext(ctx).
		Model(&model.GroupSetting{}).
		Where("group_id = ?", groupID).
		Updates(updates).Error
}

// Delete 删除群组设置
func (r *groupSettingRepositoryImpl) Delete(ctx context.Context, groupID string) error {
	return r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Delete(&model.GroupSetting{}).Error
}

// WithTx 使用事务
func (r *groupSettingRepositoryImpl) WithTx(tx *gorm.DB) GroupSettingRepository {
	return &groupSettingRepositoryImpl{db: tx}
}
