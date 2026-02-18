package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/session/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SessionRepository 会话仓库接口
type SessionRepository interface {
	// Upsert 创建或更新会话
	Upsert(ctx context.Context, session *model.Session) error
	// GetByID 根据会话ID获取会话
	GetByID(ctx context.Context, sessionID string) (*model.Session, error)
	// GetByUserAndTarget 根据用户ID和目标ID获取会话
	GetByUserAndTarget(ctx context.Context, userID, sessionType, targetID string) (*model.Session, error)
	// ListByUser 获取用户的会话列表
	ListByUser(ctx context.Context, userID string, limit int, updatedBefore *time.Time) ([]*model.Session, error)
	// Delete 删除会话
	Delete(ctx context.Context, userID, sessionID string) error
	// SetPinned 设置置顶状态
	SetPinned(ctx context.Context, userID, sessionID string, pinned bool, pinTime *time.Time) error
	// SetMuted 设置免打扰状态
	SetMuted(ctx context.Context, userID, sessionID string, muted bool) error
	// ClearUnread 清除未读数
	ClearUnread(ctx context.Context, userID, sessionID string) error
	// IncrUnread 增加未读数
	IncrUnread(ctx context.Context, userID, sessionID string, count int32) error
	// SumUnread 统计用户总未读数
	SumUnread(ctx context.Context, userID string) (int32, error)
	// WithTx 使用事务
	WithTx(tx *gorm.DB) SessionRepository
}

// sessionRepositoryImpl 会话仓库实现
type sessionRepositoryImpl struct {
	db *gorm.DB
}

// NewSessionRepository 创建会话仓库
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepositoryImpl{db: db}
}

// Upsert 创建或更新会话（冲突时更新最后消息信息）
func (r *sessionRepositoryImpl) Upsert(ctx context.Context, session *model.Session) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "session_type"}, {Name: "target_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"last_message_id",
				"last_message_content",
				"last_message_time",
				"updated_at",
			}),
		}).
		Create(session).Error
}

// GetByID 根据会话ID获取会话
func (r *sessionRepositoryImpl) GetByID(ctx context.Context, sessionID string) (*model.Session, error) {
	var session model.Session
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetByUserAndTarget 根据用户ID和目标ID获取会话
func (r *sessionRepositoryImpl) GetByUserAndTarget(ctx context.Context, userID, sessionType, targetID string) (*model.Session, error) {
	var session model.Session
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND session_type = ? AND target_id = ?", userID, sessionType, targetID).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListByUser 获取用户会话列表（按置顶+最后消息时间排序）
func (r *sessionRepositoryImpl) ListByUser(ctx context.Context, userID string, limit int, updatedBefore *time.Time) ([]*model.Session, error) {
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if updatedBefore != nil {
		q = q.Where("updated_at < ?", updatedBefore)
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var sessions []*model.Session
	err := q.Order("is_pinned DESC, COALESCE(last_message_time, created_at) DESC").
		Limit(limit).
		Find(&sessions).Error
	return sessions, err
}

// Delete 删除会话（仅删除属于该用户的会话）
func (r *sessionRepositoryImpl) Delete(ctx context.Context, userID, sessionID string) error {
	return r.db.WithContext(ctx).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Delete(&model.Session{}).Error
}

// SetPinned 设置置顶状态
func (r *sessionRepositoryImpl) SetPinned(ctx context.Context, userID, sessionID string, pinned bool, pinTime *time.Time) error {
	updates := map[string]interface{}{
		"is_pinned": pinned,
		"pin_time":  pinTime,
	}
	return r.db.WithContext(ctx).Model(&model.Session{}).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Updates(updates).Error
}

// SetMuted 设置免打扰状态
func (r *sessionRepositoryImpl) SetMuted(ctx context.Context, userID, sessionID string, muted bool) error {
	return r.db.WithContext(ctx).Model(&model.Session{}).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Update("is_muted", muted).Error
}

// ClearUnread 清除未读数
func (r *sessionRepositoryImpl) ClearUnread(ctx context.Context, userID, sessionID string) error {
	return r.db.WithContext(ctx).Model(&model.Session{}).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Update("unread_count", 0).Error
}

// IncrUnread 增加未读数
func (r *sessionRepositoryImpl) IncrUnread(ctx context.Context, userID, sessionID string, count int32) error {
	return r.db.WithContext(ctx).Model(&model.Session{}).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + ?", count)).Error
}

// SumUnread 统计用户所有未读数之和（免打扰会话不计入）
func (r *sessionRepositoryImpl) SumUnread(ctx context.Context, userID string) (int32, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.Session{}).
		Where("user_id = ? AND is_muted = false", userID).
		Select("COALESCE(SUM(unread_count), 0)").
		Scan(&total).Error
	return int32(total), err
}

// WithTx 返回使用事务的仓库实例
func (r *sessionRepositoryImpl) WithTx(tx *gorm.DB) SessionRepository {
	return &sessionRepositoryImpl{db: tx}
}
