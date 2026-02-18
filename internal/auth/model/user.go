package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID           string         `gorm:"column:id;primaryKey" json:"id"`
	Phone        *string        `gorm:"column:phone;unique" json:"phone"`
	Email        *string        `gorm:"column:email;unique" json:"email"`
	PasswordHash string         `gorm:"column:password_hash;not null" json:"-"`
	Status       int            `gorm:"column:status;not null;default:1" json:"status"` // 1-正常 2-禁用
	CreatedAt    time.Time      `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 表名
func (User) TableName() string {
	return "users"
}

// UserStatus 用户状态
const (
	UserStatusNormal   = 1 // 正常
	UserStatusDisabled = 2 // 禁用
)

// IsActive 是否激活
func (u *User) IsActive() bool {
	return u.Status == UserStatusNormal
}
