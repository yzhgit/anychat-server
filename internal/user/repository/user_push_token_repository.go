package repository

import (
	"context"

	"github.com/anychat/server/internal/user/model"
	"gorm.io/gorm"
)

// UserPushTokenRepository 推送Token仓库接口
type UserPushTokenRepository interface {
	CreateOrUpdate(ctx context.Context, token *model.UserPushToken) error
	GetByUserID(ctx context.Context, userID string) ([]*model.UserPushToken, error)
	GetByUserIDAndDeviceID(ctx context.Context, userID, deviceID string) (*model.UserPushToken, error)
	Delete(ctx context.Context, userID, deviceID string) error
}

// userPushTokenRepositoryImpl 推送Token仓库实现
type userPushTokenRepositoryImpl struct {
	db *gorm.DB
}

// NewUserPushTokenRepository 创建推送Token仓库
func NewUserPushTokenRepository(db *gorm.DB) UserPushTokenRepository {
	return &userPushTokenRepositoryImpl{db: db}
}

// CreateOrUpdate 创建或更新推送Token
func (r *userPushTokenRepositoryImpl) CreateOrUpdate(ctx context.Context, token *model.UserPushToken) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", token.UserID, token.DeviceID).
		Assign(token).
		FirstOrCreate(token).Error
}

// GetByUserID 获取用户的所有推送Token
func (r *userPushTokenRepositoryImpl) GetByUserID(ctx context.Context, userID string) ([]*model.UserPushToken, error) {
	var tokens []*model.UserPushToken
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// GetByUserIDAndDeviceID 根据用户ID和设备ID获取推送Token
func (r *userPushTokenRepositoryImpl) GetByUserIDAndDeviceID(ctx context.Context, userID, deviceID string) (*model.UserPushToken, error) {
	var token model.UserPushToken
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// Delete 删除推送Token
func (r *userPushTokenRepositoryImpl) Delete(ctx context.Context, userID, deviceID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Delete(&model.UserPushToken{}).Error
}
