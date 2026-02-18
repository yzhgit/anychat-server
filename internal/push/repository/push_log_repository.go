package repository

import (
	"github.com/anychat/server/internal/push/model"
	"gorm.io/gorm"
)

// PushTokenRow 从 user_push_tokens 表读取的推送 token 记录
type PushTokenRow struct {
	UserID   string
	DeviceID string
	Token    string   // JPush registration_id
	Platform string   // ios / android
}

// PushLogRepository 推送日志仓库接口
type PushLogRepository interface {
	Create(log *model.PushLog) error
	GetTokensByUserID(userID string) ([]*PushTokenRow, error)
	GetTokensByUserIDs(userIDs []string) (map[string][]*PushTokenRow, error)
}

type pushLogRepository struct {
	db *gorm.DB
}

// NewPushLogRepository 创建推送日志仓库
func NewPushLogRepository(db *gorm.DB) PushLogRepository {
	return &pushLogRepository{db: db}
}

// Create 创建推送日志
func (r *pushLogRepository) Create(log *model.PushLog) error {
	return r.db.Create(log).Error
}

// GetTokensByUserID 获取指定用户的所有推送 token
func (r *pushLogRepository) GetTokensByUserID(userID string) ([]*PushTokenRow, error) {
	var rows []*PushTokenRow
	err := r.db.Raw(
		`SELECT user_id, device_id, push_token AS token, platform
		   FROM user_push_tokens
		  WHERE user_id = ?`, userID,
	).Scan(&rows).Error
	return rows, err
}

// GetTokensByUserIDs 批量获取多个用户的推送 token
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
