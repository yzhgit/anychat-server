package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/group/model"
	"gorm.io/gorm"
)

// GroupJoinRequestRepository defines the join request repository interface
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

// groupJoinRequestRepositoryImpl is the join request repository implementation
type groupJoinRequestRepositoryImpl struct {
	db *gorm.DB
}

// NewGroupJoinRequestRepository creates a new join request repository
func NewGroupJoinRequestRepository(db *gorm.DB) GroupJoinRequestRepository {
	return &groupJoinRequestRepositoryImpl{db: db}
}

// Create creates a join request
func (r *groupJoinRequestRepositoryImpl) Create(ctx context.Context, request *model.GroupJoinRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

// GetByID gets join request by ID
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

// UpdateStatus updates request status
func (r *groupJoinRequestRepositoryImpl) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.GroupJoinRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// GetPendingRequestsByGroup gets pending requests for group
func (r *groupJoinRequestRepositoryImpl) GetPendingRequestsByGroup(ctx context.Context, groupID string) ([]*model.GroupJoinRequest, error) {
	var requests []*model.GroupJoinRequest
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND status = ?", groupID, model.JoinRequestStatusPending).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// GetPendingRequestsByUser gets pending requests for user
func (r *groupJoinRequestRepositoryImpl) GetPendingRequestsByUser(ctx context.Context, userID string) ([]*model.GroupJoinRequest, error) {
	var requests []*model.GroupJoinRequest
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, model.JoinRequestStatusPending).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// GetRequestsByGroup gets request list for group (filterable by status)
func (r *groupJoinRequestRepositoryImpl) GetRequestsByGroup(ctx context.Context, groupID string, status *string) ([]*model.GroupJoinRequest, error) {
	var requests []*model.GroupJoinRequest
	query := r.db.WithContext(ctx).Where("group_id = ?", groupID)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Order("created_at DESC").Find(&requests).Error
	return requests, err
}

// GetExistingRequest gets existing pending request (prevents duplicate requests)
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

// WithTx uses transaction
func (r *groupJoinRequestRepositoryImpl) WithTx(tx *gorm.DB) GroupJoinRequestRepository {
	return &groupJoinRequestRepositoryImpl{db: tx}
}
