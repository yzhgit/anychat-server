package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/user/model"
	"gorm.io/gorm"
)

// UserQRCodeRepository 用户二维码仓库接口
type UserQRCodeRepository interface {
	Create(ctx context.Context, qrcode *model.UserQRCode) error
	GetByToken(ctx context.Context, token string) (*model.UserQRCode, error)
	GetLatestByUserID(ctx context.Context, userID string) (*model.UserQRCode, error)
	DeleteExpired(ctx context.Context) error
}

// userQRCodeRepositoryImpl 用户二维码仓库实现
type userQRCodeRepositoryImpl struct {
	db *gorm.DB
}

// NewUserQRCodeRepository 创建用户二维码仓库
func NewUserQRCodeRepository(db *gorm.DB) UserQRCodeRepository {
	return &userQRCodeRepositoryImpl{db: db}
}

// Create 创建二维码
func (r *userQRCodeRepositoryImpl) Create(ctx context.Context, qrcode *model.UserQRCode) error {
	return r.db.WithContext(ctx).Create(qrcode).Error
}

// GetByToken 根据Token获取二维码
func (r *userQRCodeRepositoryImpl) GetByToken(ctx context.Context, token string) (*model.UserQRCode, error) {
	var qrcode model.UserQRCode
	err := r.db.WithContext(ctx).
		Where("qrcode_token = ?", token).
		First(&qrcode).Error
	if err != nil {
		return nil, err
	}
	return &qrcode, nil
}

// GetLatestByUserID 获取用户最新的二维码
func (r *userQRCodeRepositoryImpl) GetLatestByUserID(ctx context.Context, userID string) (*model.UserQRCode, error) {
	var qrcode model.UserQRCode
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&qrcode).Error
	if err != nil {
		return nil, err
	}
	return &qrcode, nil
}

// DeleteExpired 删除过期的二维码
func (r *userQRCodeRepositoryImpl) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&model.UserQRCode{}).Error
}
