package repository

import (
	"context"

	"github.com/anychat/server/internal/auth/model"
	"gorm.io/gorm"
)

// UserSessionRepository 用户会话仓库接口
type UserSessionRepository interface {
	Create(ctx context.Context, session *model.UserSession) error
	GetByAccessToken(ctx context.Context, accessToken string) (*model.UserSession, error)
	GetByRefreshToken(ctx context.Context, refreshToken string) (*model.UserSession, error)
	GetByUserIDAndDeviceID(ctx context.Context, userID, deviceID string) (*model.UserSession, error)
	Update(ctx context.Context, session *model.UserSession) error
	DeleteByUserIDAndDeviceID(ctx context.Context, userID, deviceID string) error
	DeleteByUserID(ctx context.Context, userID string) error
}

// userSessionRepositoryImpl 用户会话仓库实现
type userSessionRepositoryImpl struct {
	db *gorm.DB
}

// NewUserSessionRepository 创建用户会话仓库
func NewUserSessionRepository(db *gorm.DB) UserSessionRepository {
	return &userSessionRepositoryImpl{db: db}
}

// Create 创建会话
func (r *userSessionRepositoryImpl) Create(ctx context.Context, session *model.UserSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// GetByAccessToken 根据AccessToken获取会话
func (r *userSessionRepositoryImpl) GetByAccessToken(ctx context.Context, accessToken string) (*model.UserSession, error) {
	var session model.UserSession
	err := r.db.WithContext(ctx).
		Where("access_token = ?", accessToken).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetByRefreshToken 根据RefreshToken获取会话
func (r *userSessionRepositoryImpl) GetByRefreshToken(ctx context.Context, refreshToken string) (*model.UserSession, error) {
	var session model.UserSession
	err := r.db.WithContext(ctx).
		Where("refresh_token = ?", refreshToken).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetByUserIDAndDeviceID 根据用户ID和设备ID获取会话
func (r *userSessionRepositoryImpl) GetByUserIDAndDeviceID(ctx context.Context, userID, deviceID string) (*model.UserSession, error) {
	var session model.UserSession
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// Update 更新会话
func (r *userSessionRepositoryImpl) Update(ctx context.Context, session *model.UserSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

// DeleteByUserIDAndDeviceID 删除指定设备的会话
func (r *userSessionRepositoryImpl) DeleteByUserIDAndDeviceID(ctx context.Context, userID, deviceID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Delete(&model.UserSession{}).Error
}

// DeleteByUserID 删除用户的所有会话
func (r *userSessionRepositoryImpl) DeleteByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.UserSession{}).Error
}
