package model

import "time"

// AdminUser 管理员账户
type AdminUser struct {
	ID           string     `gorm:"column:id;primaryKey"`
	Username     string     `gorm:"column:username;uniqueIndex;not null"`
	PasswordHash string     `gorm:"column:password_hash;not null"`
	Email        string     `gorm:"column:email"`
	Role         string     `gorm:"column:role;not null;default:admin"`
	Status       int8       `gorm:"column:status;not null;default:1"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (AdminUser) TableName() string { return "admin_users" }

// IsActive 是否有效
func (a *AdminUser) IsActive() bool { return a.Status == 1 }
