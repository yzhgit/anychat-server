package model

import (
	"time"
)

// UserPushToken push token model
type UserPushToken struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    string    `gorm:"column:user_id;not null" json:"userId"`
	DeviceID  string    `gorm:"column:device_id;not null" json:"deviceId"`
	PushToken string    `gorm:"column:push_token;not null" json:"pushToken"`
	Platform  string    `gorm:"column:platform;not null" json:"platform"` // iOS/Android
	CreatedAt time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

// TableName returns table name
func (UserPushToken) TableName() string {
	return "user_push_tokens"
}
