package repository

import (
	"context"

	"github.com/anychat/server/internal/user/model"
	"gorm.io/gorm"
)

// UserProfileRepository 用户资料仓库接口
type UserProfileRepository interface {
	Create(ctx context.Context, profile *model.UserProfile) error
	GetByUserID(ctx context.Context, userID string) (*model.UserProfile, error)
	Update(ctx context.Context, profile *model.UserProfile) error
	UpdateQRCode(ctx context.Context, userID, qrcodeURL string) error
	CheckNicknameExists(ctx context.Context, nickname string, excludeUserID string) (bool, error)
	SearchByKeyword(ctx context.Context, keyword string, limit, offset int) ([]*model.UserProfile, int64, error)
}

// userProfileRepositoryImpl 用户资料仓库实现
type userProfileRepositoryImpl struct {
	db *gorm.DB
}

// NewUserProfileRepository 创建用户资料仓库
func NewUserProfileRepository(db *gorm.DB) UserProfileRepository {
	return &userProfileRepositoryImpl{db: db}
}

// Create 创建用户资料
func (r *userProfileRepositoryImpl) Create(ctx context.Context, profile *model.UserProfile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

// GetByUserID 根据用户ID获取资料
func (r *userProfileRepositoryImpl) GetByUserID(ctx context.Context, userID string) (*model.UserProfile, error) {
	var profile model.UserProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Update 更新用户资料
func (r *userProfileRepositoryImpl) Update(ctx context.Context, profile *model.UserProfile) error {
	return r.db.WithContext(ctx).Save(profile).Error
}

// UpdateQRCode 更新二维码
func (r *userProfileRepositoryImpl) UpdateQRCode(ctx context.Context, userID, qrcodeURL string) error {
	return r.db.WithContext(ctx).
		Model(&model.UserProfile{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"qrcode_url":        qrcodeURL,
			"qrcode_updated_at": gorm.Expr("NOW()"),
		}).Error
}

// CheckNicknameExists 检查昵称是否存在
func (r *userProfileRepositoryImpl) CheckNicknameExists(ctx context.Context, nickname string, excludeUserID string) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.UserProfile{}).Where("nickname = ?", nickname)

	if excludeUserID != "" {
		query = query.Where("user_id != ?", excludeUserID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// SearchByKeyword 根据关键词搜索用户
func (r *userProfileRepositoryImpl) SearchByKeyword(ctx context.Context, keyword string, limit, offset int) ([]*model.UserProfile, int64, error) {
	var profiles []*model.UserProfile
	var total int64

	// 搜索昵称
	query := r.db.WithContext(ctx).Model(&model.UserProfile{}).
		Where("nickname LIKE ?", "%"+keyword+"%")

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if err := query.Limit(limit).Offset(offset).Find(&profiles).Error; err != nil {
		return nil, 0, err
	}

	return profiles, total, nil
}
