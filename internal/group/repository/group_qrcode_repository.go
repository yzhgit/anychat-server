package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/group/model"
	"gorm.io/gorm"
)

// GroupQRCodeRepository 群二维码仓库接口
type GroupQRCodeRepository interface {
	Create(ctx context.Context, qrcode *model.GroupQRCode) error
	GetActiveByGroupID(ctx context.Context, groupID string) (*model.GroupQRCode, error)
	GetByToken(ctx context.Context, token string) (*model.GroupQRCode, error)
	InvalidateByGroupID(ctx context.Context, groupID string) error
	UpdateExpireAt(ctx context.Context, token string, expireAt time.Time) error
	WithTx(tx *gorm.DB) GroupQRCodeRepository
}

type groupQRCodeRepositoryImpl struct {
	db *gorm.DB
}

func NewGroupQRCodeRepository(db *gorm.DB) GroupQRCodeRepository {
	return &groupQRCodeRepositoryImpl{db: db}
}

// Create 创建二维码记录
func (r *groupQRCodeRepositoryImpl) Create(ctx context.Context, qrcode *model.GroupQRCode) error {
	return r.db.WithContext(ctx).Create(qrcode).Error
}

// GetActiveByGroupID 获取群的当前有效二维码
func (r *groupQRCodeRepositoryImpl) GetActiveByGroupID(ctx context.Context, groupID string) (*model.GroupQRCode, error) {
	var qr model.GroupQRCode
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND is_active = true", groupID).
		Order("created_at DESC").
		First(&qr).Error
	if err != nil {
		return nil, err
	}
	return &qr, nil
}

// GetByToken 根据 token 查询二维码
func (r *groupQRCodeRepositoryImpl) GetByToken(ctx context.Context, token string) (*model.GroupQRCode, error) {
	var qr model.GroupQRCode
	err := r.db.WithContext(ctx).
		Where("token = ?", token).
		First(&qr).Error
	if err != nil {
		return nil, err
	}
	return &qr, nil
}

// InvalidateByGroupID 将群内所有活跃二维码标记为失效
func (r *groupQRCodeRepositoryImpl) InvalidateByGroupID(ctx context.Context, groupID string) error {
	return r.db.WithContext(ctx).
		Model(&model.GroupQRCode{}).
		Where("group_id = ? AND is_active = true", groupID).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": time.Now(),
		}).Error
}

// UpdateExpireAt 续期二维码
func (r *groupQRCodeRepositoryImpl) UpdateExpireAt(ctx context.Context, token string, expireAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.GroupQRCode{}).
		Where("token = ?", token).
		Updates(map[string]interface{}{
			"expire_at":  expireAt,
			"updated_at": time.Now(),
		}).Error
}

func (r *groupQRCodeRepositoryImpl) WithTx(tx *gorm.DB) GroupQRCodeRepository {
	return &groupQRCodeRepositoryImpl{db: tx}
}
