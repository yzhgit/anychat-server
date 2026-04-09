package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/user/model"
	"gorm.io/gorm"
)

// UserQRCodeRepository user QR code repository interface
type UserQRCodeRepository interface {
	Create(ctx context.Context, qrcode *model.UserQRCode) error
	GetByToken(ctx context.Context, token string) (*model.UserQRCode, error)
	GetLatestByUserID(ctx context.Context, userID string) (*model.UserQRCode, error)
	DeleteExpired(ctx context.Context) error
}

// userQRCodeRepositoryImpl user QR code repository implementation
type userQRCodeRepositoryImpl struct {
	db *gorm.DB
}

// NewUserQRCodeRepository creates user QR code repository
func NewUserQRCodeRepository(db *gorm.DB) UserQRCodeRepository {
	return &userQRCodeRepositoryImpl{db: db}
}

// Create creates QR code
func (r *userQRCodeRepositoryImpl) Create(ctx context.Context, qrcode *model.UserQRCode) error {
	return r.db.WithContext(ctx).Create(qrcode).Error
}

// GetByToken retrieves QR code by token
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

// GetLatestByUserID retrieves latest QR code for user
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

// DeleteExpired deletes expired QR codes
func (r *userQRCodeRepositoryImpl) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&model.UserQRCode{}).Error
}
