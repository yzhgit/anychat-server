package model

import (
	"encoding/json"
	"time"
)

// AuditLog 操作审计日志
type AuditLog struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement"`
	AdminID      string          `gorm:"column:admin_id"`
	Action       string          `gorm:"column:action;not null"`
	ResourceType string          `gorm:"column:resource_type"`
	ResourceID   string          `gorm:"column:resource_id"`
	Details      json.RawMessage `gorm:"column:details;type:jsonb"`
	IPAddress    string          `gorm:"column:ip_address"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime"`
}

func (AuditLog) TableName() string { return "audit_logs" }
