package repository

import (
	"context"

	"github.com/anychat/server/internal/user/model"
	"gorm.io/gorm"
)

// UserSettingsRepository 用户设置仓库接口
type UserSettingsRepository interface {
	Create(ctx context.Context, settings *model.UserSettings) error
	GetByUserID(ctx context.Context, userID string) (*model.UserSettings, error)
	Update(ctx context.Context, settings *model.UserSettings) error
}

// userSettingsRepositoryImpl 用户设置仓库实现
type userSettingsRepositoryImpl struct {
	db *gorm.DB
}

// NewUserSettingsRepository 创建用户设置仓库
func NewUserSettingsRepository(db *gorm.DB) UserSettingsRepository {
	return &userSettingsRepositoryImpl{db: db}
}

// Create 创建用户设置
func (r *userSettingsRepositoryImpl) Create(ctx context.Context, settings *model.UserSettings) error {
	return r.db.WithContext(ctx).Create(settings).Error
}

// GetByUserID 根据用户ID获取设置
func (r *userSettingsRepositoryImpl) GetByUserID(ctx context.Context, userID string) (*model.UserSettings, error) {
	var settings model.UserSettings
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&settings).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// Update 更新用户设置
func (r *userSettingsRepositoryImpl) Update(ctx context.Context, settings *model.UserSettings) error {
	return r.db.WithContext(ctx).Save(settings).Error
}
