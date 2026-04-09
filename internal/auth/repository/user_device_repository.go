package repository

import (
	"context"

	"github.com/anychat/server/internal/auth/model"
	"gorm.io/gorm"
)

// UserDeviceRepository user device repository interface
type UserDeviceRepository interface {
	Create(ctx context.Context, device *model.UserDevice) error
	GetByUserIDAndDeviceID(ctx context.Context, userID, deviceID string) (*model.UserDevice, error)
	GetByUserID(ctx context.Context, userID string) ([]*model.UserDevice, error)
	GetByUserIDAndDeviceType(ctx context.Context, userID, deviceType string) ([]*model.UserDevice, error)
	Update(ctx context.Context, device *model.UserDevice) error
	UpdateLastLogin(ctx context.Context, userID, deviceID, ip string) error
}

// userDeviceRepositoryImpl user device repository implementation
type userDeviceRepositoryImpl struct {
	db *gorm.DB
}

// NewUserDeviceRepository creates user device repository
func NewUserDeviceRepository(db *gorm.DB) UserDeviceRepository {
	return &userDeviceRepositoryImpl{db: db}
}

// Create creates device record
func (r *userDeviceRepositoryImpl) Create(ctx context.Context, device *model.UserDevice) error {
	return r.db.WithContext(ctx).Create(device).Error
}

// GetByUserIDAndDeviceID gets device by user ID and device ID
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

// GetByUserID gets all devices for user
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

// GetByUserIDAndDeviceType gets devices by user ID and device type
func (r *userDeviceRepositoryImpl) GetByUserIDAndDeviceType(ctx context.Context, userID, deviceType string) ([]*model.UserDevice, error) {
	var devices []*model.UserDevice
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND device_type = ?", userID, deviceType).
		Order("last_login_at DESC").
		Find(&devices).Error
	if err != nil {
		return nil, err
	}
	return devices, nil
}

// Update updates device record
func (r *userDeviceRepositoryImpl) Update(ctx context.Context, device *model.UserDevice) error {
	return r.db.WithContext(ctx).Save(device).Error
}

// UpdateLastLogin updates last login info
func (r *userDeviceRepositoryImpl) UpdateLastLogin(ctx context.Context, userID, deviceID, ip string) error {
	return r.db.WithContext(ctx).
		Model(&model.UserDevice{}).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Updates(map[string]interface{}{
			"last_login_at": gorm.Expr("NOW()"),
			"last_login_ip": ip,
		}).Error
}
