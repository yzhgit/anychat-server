package repository

import (
	"github.com/anychat/server/internal/push/model"
	"gorm.io/gorm"
)

// PushTokenRow push token record read from user_push_tokens table
type PushTokenRow struct {
	UserID   string
	DeviceID string
	Token    string // JPush registration_id
	Platform string // ios / android
}

// PushLogRepository push log repository interface
type PushLogRepository interface {
	Create(log *model.PushLog) error
	GetTokensByUserID(userID string) ([]*PushTokenRow, error)
	GetTokensByUserIDs(userIDs []string) (map[string][]*PushTokenRow, error)
}

type pushLogRepository struct {
	db *gorm.DB
}

// NewPushLogRepository creates push log repository
func NewPushLogRepository(db *gorm.DB) PushLogRepository {
	return &pushLogRepository{db: db}
}

// Create creates push log
func (r *pushLogRepository) Create(log *model.PushLog) error {
	return r.db.Create(log).Error
}

// GetTokensByUserID retrieves all push tokens for specified user
func (r *pushLogRepository) GetTokensByUserID(userID string) ([]*PushTokenRow, error) {
	var rows []*PushTokenRow
	err := r.db.Raw(
		`SELECT user_id, device_id, push_token AS token, platform
		   FROM user_push_tokens
		  WHERE user_id = ?`, userID,
	).Scan(&rows).Error
	return rows, err
}

// GetTokensByUserIDs retrieves push tokens for multiple users in batch
func (r *pushLogRepository) GetTokensByUserIDs(userIDs []string) (map[string][]*PushTokenRow, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	var rows []*PushTokenRow
	err := r.db.Raw(
		`SELECT user_id, device_id, push_token AS token, platform
		   FROM user_push_tokens
		  WHERE user_id IN ?`, userIDs,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string][]*PushTokenRow, len(userIDs))
	for _, row := range rows {
		result[row.UserID] = append(result[row.UserID], row)
	}
	return result, nil
}
