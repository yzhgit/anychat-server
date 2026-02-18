package repository

import (
	"context"

	"github.com/anychat/server/internal/friend/model"
	"gorm.io/gorm"
)

// FriendRequestRepository 好友申请仓库接口
type FriendRequestRepository interface {
	Create(ctx context.Context, request *model.FriendRequest) error
	GetByID(ctx context.Context, id int64) (*model.FriendRequest, error)
	GetPendingRequest(ctx context.Context, fromUserID, toUserID string) (*model.FriendRequest, error)
	GetReceivedRequests(ctx context.Context, userID string) ([]*model.FriendRequest, error)
	GetSentRequests(ctx context.Context, userID string) ([]*model.FriendRequest, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	Update(ctx context.Context, request *model.FriendRequest) error
	WithTx(tx *gorm.DB) FriendRequestRepository
}

// friendRequestRepositoryImpl 好友申请仓库实现
type friendRequestRepositoryImpl struct {
	db *gorm.DB
}

// NewFriendRequestRepository 创建好友申请仓库
func NewFriendRequestRepository(db *gorm.DB) FriendRequestRepository {
	return &friendRequestRepositoryImpl{db: db}
}

// Create 创建好友申请
func (r *friendRequestRepositoryImpl) Create(ctx context.Context, request *model.FriendRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

// GetByID 根据ID获取好友申请
func (r *friendRequestRepositoryImpl) GetByID(ctx context.Context, id int64) (*model.FriendRequest, error) {
	var request model.FriendRequest
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// GetPendingRequest 获取待处理的好友申请
func (r *friendRequestRepositoryImpl) GetPendingRequest(ctx context.Context, fromUserID, toUserID string) (*model.FriendRequest, error) {
	var request model.FriendRequest
	err := r.db.WithContext(ctx).
		Where("from_user_id = ? AND to_user_id = ? AND status = ?", fromUserID, toUserID, model.FriendRequestStatusPending).
		Order("created_at DESC").
		First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// GetReceivedRequests 获取收到的好友申请
func (r *friendRequestRepositoryImpl) GetReceivedRequests(ctx context.Context, userID string) ([]*model.FriendRequest, error) {
	var requests []*model.FriendRequest
	err := r.db.WithContext(ctx).
		Where("to_user_id = ?", userID).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// GetSentRequests 获取发送的好友申请
func (r *friendRequestRepositoryImpl) GetSentRequests(ctx context.Context, userID string) ([]*model.FriendRequest, error) {
	var requests []*model.FriendRequest
	err := r.db.WithContext(ctx).
		Where("from_user_id = ?", userID).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// UpdateStatus 更新申请状态
func (r *friendRequestRepositoryImpl) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.FriendRequest{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// Update 更新好友申请
func (r *friendRequestRepositoryImpl) Update(ctx context.Context, request *model.FriendRequest) error {
	return r.db.WithContext(ctx).Save(request).Error
}

// WithTx 使用事务
func (r *friendRequestRepositoryImpl) WithTx(tx *gorm.DB) FriendRequestRepository {
	return &friendRequestRepositoryImpl{db: tx}
}
