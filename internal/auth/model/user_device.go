package model

import (
	"time"
)

// UserDevice 用户设备模型
type UserDevice struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID        string     `gorm:"column:user_id;not null" json:"userId"`
	DeviceID      string     `gorm:"column:device_id;not null" json:"deviceId"`
	DeviceType    string     `gorm:"column:device_type;not null" json:"deviceType"`
	ClientVersion string     `gorm:"column:client_version" json:"clientVersion"`
	LastLoginAt   *time.Time `gorm:"column:last_login_at" json:"lastLoginAt"`
	LastLoginIP   string     `gorm:"column:last_login_ip" json:"lastLoginIp"`
	CreatedAt     time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt     time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

// TableName 表名
func (UserDevice) TableName() string {
	return "user_devices"
}
