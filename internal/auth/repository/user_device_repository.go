package repository

import (
	"context"

	"github.com/anychat/server/internal/auth/model"
	"gorm.io/gorm"
)

// UserDeviceRepository 用户设备仓库接口
type UserDeviceRepository interface {
	Create(ctx context.Context, device *model.UserDevice) error
	GetByUserIDAndDeviceID(ctx context.Context, userID, deviceID string) (*model.UserDevice, error)
	GetByUserID(ctx context.Context, userID string) ([]*model.UserDevice, error)
	Update(ctx context.Context, device *model.UserDevice) error
	UpdateLastLogin(ctx context.Context, userID, deviceID, ip string) error
}

// userDeviceRepositoryImpl 用户设备仓库实现
type userDeviceRepositoryImpl struct {
	db *gorm.DB
}

// NewUserDeviceRepository 创建用户设备仓库
func NewUserDeviceRepository(db *gorm.DB) UserDeviceRepository {
	return &userDeviceRepositoryImpl{db: db}
}

// Create 创建设备记录
func (r *userDeviceRepositoryImpl) Create(ctx context.Context, device *model.UserDevice) error {
	return r.db.WithContext(ctx).Create(device).Error
}

// GetByUserIDAndDeviceID 根据用户ID和设备ID获取设备
func (r *userDeviceRepositoryImpl) GetByUserIDAndDeviceID(ctx context.Context, userID, deviceID string) (*model.UserDevice, error) {
	var device model.UserDevice
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetByUserID 获取用户的所有设备
func (r *userDeviceRepositoryImpl) GetByUserID(ctx context.Context, userID string) ([]*model.UserDevice, error) {
	var devices []*model.UserDevice
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_login_at DESC").
		Find(&devices).Error
	if err != nil {
		return nil, err
	}
	return devices, nil
}

// Update 更新设备记录
func (r *userDeviceRepositoryImpl) Update(ctx context.Context, device *model.UserDevice) error {
	return r.db.WithContext(ctx).Save(device).Error
}

// UpdateLastLogin 更新最后登录信息
func (r *userDeviceRepositoryImpl) UpdateLastLogin(ctx context.Context, userID, deviceID, ip string) error {
	return r.db.WithContext(ctx).
		Model(&model.UserDevice{}).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Updates(map[string]interface{}{
			"last_login_at": gorm.Expr("NOW()"),
			"last_login_ip": ip,
		}).Error
}
