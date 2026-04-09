package model

import "time"

// VerificationCode verification code model
type VerificationCode struct {
	ID                int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CodeID            string     `gorm:"column:code_id;not null;uniqueIndex" json:"codeId"`
	Target            string     `gorm:"column:target;not null;index" json:"target"`
	TargetType        string     `gorm:"column:target_type;not null" json:"targetType"`
	CodeHash          string     `gorm:"column:code_hash;not null" json:"-"`
	Purpose           string     `gorm:"column:purpose;not null" json:"purpose"`
	ExpiresAt         time.Time  `gorm:"column:expires_at;not null" json:"expiresAt"`
	VerifiedAt        *time.Time `gorm:"column:verified_at" json:"verifiedAt"`
	Status            string     `gorm:"column:status;not null;default:pending" json:"status"`
	SendIP            string     `gorm:"column:send_ip" json:"sendIp"`
	SendDeviceID      string     `gorm:"column:send_device_id" json:"sendDeviceId"`
	AttemptCount      int        `gorm:"column:attempt_count;not null;default:0" json:"attemptCount"`
	Provider          string     `gorm:"column:provider" json:"provider"`
	ProviderMessageID string     `gorm:"column:provider_message_id" json:"providerMessageId"`
	CreatedAt         time.Time  `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (VerificationCode) TableName() string {
	return "verification_codes"
}

const (
	CodeStatusPending   = "pending"
	CodeStatusVerified  = "verified"
	CodeStatusExpired   = "expired"
	CodeStatusLocked    = "locked"
	CodeStatusCancelled = "cancelled"
)

const (
	TargetTypeSMS   = "sms"
	TargetTypeEmail = "email"
)

const (
	PurposeRegister      = "register"
	PurposeLogin         = "login"
	PurposeResetPassword = "reset_password"
	PurposeBindPhone     = "bind_phone"
	PurposeChangePhone   = "change_phone"
	PurposeBindEmail     = "bind_email"
	PurposeChangeEmail   = "change_email"
)

func (v *VerificationCode) IsExpired() bool {
	return time.Now().After(v.ExpiresAt)
}

func (v *VerificationCode) IsPending() bool {
	return v.Status == CodeStatusPending
}

func (v *VerificationCode) IsVerified() bool {
	return v.Status == CodeStatusVerified
}

func (v *VerificationCode) IsLocked() bool {
	return v.Status == CodeStatusLocked
}
