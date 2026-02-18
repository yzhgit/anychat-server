package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/group/model"
	"gorm.io/gorm"
)

// GroupJoinRequestRepository 入群申请仓库接口
type GroupJoinRequestRepository interface {
	Create(ctx context.Context, request *model.GroupJoinRequest) error
	GetByID(ctx context.Context, id int64) (*model.GroupJoinRequest, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	GetPendingRequestsByGroup(ctx context.Context, groupID string) ([]*model.GroupJoinRequest, error)
	GetPendingRequestsByUser(ctx context.Context, userID string) ([]*model.GroupJoinRequest, error)
	GetRequestsByGroup(ctx context.Context, groupID string, status *string) ([]*model.GroupJoinRequest, error)
	GetExistingRequest(ctx context.Context, groupID, userID string) (*model.GroupJoinRequest, error)
	WithTx(tx *gorm.DB) GroupJoinRequestRepository
}

// groupJoinRequestRepositoryImpl 入群申请仓库实现
type groupJoinRequestRepositoryImpl struct {
	db *gorm.DB
}

// NewGroupJoinRequestRepository 创建入群申请仓库
func NewGroupJoinRequestRepository(db *gorm.DB) GroupJoinRequestRepository {
	return &groupJoinRequestRepositoryImpl{db: db}
}

// Create 创建入群申请
func (r *groupJoinRequestRepositoryImpl) Create(ctx context.Context, request *model.GroupJoinRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

// GetByID 根据ID获取入群申请
func (r *groupJoinRequestRepositoryImpl) GetByID(ctx context.Context, id int64) (*model.GroupJoinRequest, error) {
	var request model.GroupJoinRequest
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// UpdateStatus 更新申请状态
func (r *groupJoinRequestRepositoryImpl) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.GroupJoinRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// GetPendingRequestsByGroup 获取群组的待处理申请
func (r *groupJoinRequestRepositoryImpl) GetPendingRequestsByGroup(ctx context.Context, groupID string) ([]*model.GroupJoinRequest, error) {
	var requests []*model.GroupJoinRequest
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND status = ?", groupID, model.JoinRequestStatusPending).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// GetPendingRequestsByUser 获取用户的待处理申请
func (r *groupJoinRequestRepositoryImpl) GetPendingRequestsByUser(ctx context.Context, userID string) ([]*model.GroupJoinRequest, error) {
	var requests []*model.GroupJoinRequest
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, model.JoinRequestStatusPending).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// GetRequestsByGroup 获取群组的申请列表（可按状态过滤）
func (r *groupJoinRequestRepositoryImpl) GetRequestsByGroup(ctx context.Context, groupID string, status *string) ([]*model.GroupJoinRequest, error) {
	var requests []*model.GroupJoinRequest
	query := r.db.WithContext(ctx).Where("group_id = ?", groupID)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Order("created_at DESC").Find(&requests).Error
	return requests, err
}

// GetExistingRequest 获取已存在的待处理申请（防止重复申请）
func (r *groupJoinRequestRepositoryImpl) GetExistingRequest(ctx context.Context, groupID, userID string) (*model.GroupJoinRequest, error) {
	var request model.GroupJoinRequest
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ? AND status = ?", groupID, userID, model.JoinRequestStatusPending).
		First(&request).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &request, nil
}

// WithTx 使用事务
func (r *groupJoinRequestRepositoryImpl) WithTx(tx *gorm.DB) GroupJoinRequestRepository {
	return &groupJoinRequestRepositoryImpl{db: tx}
}
