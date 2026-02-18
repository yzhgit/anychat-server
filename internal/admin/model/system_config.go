package model

import "time"

// SystemConfig 系统配置（键值对）
type SystemConfig struct {
	Key         string    `gorm:"column:key;primaryKey"`
	Value       string    `gorm:"column:value;not null;default:''"`
	Description string    `gorm:"column:description"`
	UpdatedBy   string    `gorm:"column:updated_by"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (SystemConfig) TableName() string { return "system_configs" }
