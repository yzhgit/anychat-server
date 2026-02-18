package repository

import (
	"github.com/anychat/server/internal/admin/model"
	"gorm.io/gorm"
)

// AdminUserRepository 管理员用户仓库
type AdminUserRepository interface {
	Create(user *model.AdminUser) error
	GetByID(id string) (*model.AdminUser, error)
	GetByUsername(username string) (*model.AdminUser, error)
	Update(user *model.AdminUser) error
	List(page, pageSize int) ([]*model.AdminUser, int64, error)
	UpdateLastLogin(id string) error
}

type adminUserRepository struct {
	db *gorm.DB
}

func NewAdminUserRepository(db *gorm.DB) AdminUserRepository {
	return &adminUserRepository{db: db}
}

func (r *adminUserRepository) Create(user *model.AdminUser) error {
	return r.db.Create(user).Error
}

func (r *adminUserRepository) GetByID(id string) (*model.AdminUser, error) {
	var user model.AdminUser
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *adminUserRepository) GetByUsername(username string) (*model.AdminUser, error) {
	var user model.AdminUser
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *adminUserRepository) Update(user *model.AdminUser) error {
	return r.db.Save(user).Error
}

func (r *adminUserRepository) List(page, pageSize int) ([]*model.AdminUser, int64, error) {
	var users []*model.AdminUser
	var total int64
	offset := (page - 1) * pageSize
	if err := r.db.Model(&model.AdminUser{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *adminUserRepository) UpdateLastLogin(id string) error {
	return r.db.Model(&model.AdminUser{}).Where("id = ?", id).
		Update("last_login_at", gorm.Expr("NOW()")).Error
}

// AuditLogRepository 审计日志仓库
type AuditLogRepository interface {
	Create(log *model.AuditLog) error
	List(adminID, action string, page, pageSize int) ([]*model.AuditLog, int64, error)
}

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(log *model.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *auditLogRepository) List(adminID, action string, page, pageSize int) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64
	offset := (page - 1) * pageSize
	q := r.db.Model(&model.AuditLog{})
	if adminID != "" {
		q = q.Where("admin_id = ?", adminID)
	}
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// SystemConfigRepository 系统配置仓库
type SystemConfigRepository interface {
	GetAll() ([]*model.SystemConfig, error)
	GetByKey(key string) (*model.SystemConfig, error)
	Set(cfg *model.SystemConfig) error
}

type systemConfigRepository struct {
	db *gorm.DB
}

func NewSystemConfigRepository(db *gorm.DB) SystemConfigRepository {
	return &systemConfigRepository{db: db}
}

func (r *systemConfigRepository) GetAll() ([]*model.SystemConfig, error) {
	var configs []*model.SystemConfig
	err := r.db.Order("key").Find(&configs).Error
	return configs, err
}

func (r *systemConfigRepository) GetByKey(key string) (*model.SystemConfig, error) {
	var cfg model.SystemConfig
	err := r.db.Where("key = ?", key).First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *systemConfigRepository) Set(cfg *model.SystemConfig) error {
	return r.db.Save(cfg).Error
}
