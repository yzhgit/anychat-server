package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/friend/model"
	"gorm.io/gorm"
)

// FriendshipRepository 好友关系仓库接口
type FriendshipRepository interface {
	Create(ctx context.Context, friendship *model.Friendship) error
	CreateBatch(ctx context.Context, friendships []*model.Friendship) error
	GetByUserAndFriend(ctx context.Context, userID, friendID string) (*model.Friendship, error)
	GetFriendList(ctx context.Context, userID string) ([]*model.Friendship, error)
	GetFriendListByUpdateTime(ctx context.Context, userID string, lastUpdateTime time.Time) ([]*model.Friendship, error)
	Update(ctx context.Context, friendship *model.Friendship) error
	UpdateRemark(ctx context.Context, userID, friendID, remark string) error
	Delete(ctx context.Context, userID, friendID string) error
	DeleteBidirectional(ctx context.Context, userID, friendID string) error
	IsFriend(ctx context.Context, userID, friendID string) (bool, error)
	WithTx(tx *gorm.DB) FriendshipRepository
}

// friendshipRepositoryImpl 好友关系仓库实现
type friendshipRepositoryImpl struct {
	db *gorm.DB
}

// NewFriendshipRepository 创建好友关系仓库
func NewFriendshipRepository(db *gorm.DB) FriendshipRepository {
	return &friendshipRepositoryImpl{db: db}
}

// Create 创建好友关系
func (r *friendshipRepositoryImpl) Create(ctx context.Context, friendship *model.Friendship) error {
	return r.db.WithContext(ctx).Create(friendship).Error
}

// CreateBatch 批量创建好友关系（用于双向创建）
func (r *friendshipRepositoryImpl) CreateBatch(ctx context.Context, friendships []*model.Friendship) error {
	return r.db.WithContext(ctx).Create(friendships).Error
}

// GetByUserAndFriend 根据用户ID和好友ID获取关系
func (r *friendshipRepositoryImpl) GetByUserAndFriend(ctx context.Context, userID, friendID string) (*model.Friendship, error) {
	var friendship model.Friendship
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND friend_id = ? AND status = ?", userID, friendID, model.FriendshipStatusNormal).
		First(&friendship).Error
	if err != nil {
		return nil, err
	}
	return &friendship, nil
}

// GetFriendList 获取好友列表
func (r *friendshipRepositoryImpl) GetFriendList(ctx context.Context, userID string) ([]*model.Friendship, error) {
	var friendships []*model.Friendship
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, model.FriendshipStatusNormal).
		Order("updated_at DESC").
		Find(&friendships).Error
	return friendships, err
}

// GetFriendListByUpdateTime 根据更新时间获取好友列表（增量同步）
func (r *friendshipRepositoryImpl) GetFriendListByUpdateTime(ctx context.Context, userID string, lastUpdateTime time.Time) ([]*model.Friendship, error) {
	var friendships []*model.Friendship
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND updated_at > ?", userID, lastUpdateTime).
		Order("updated_at DESC").
		Find(&friendships).Error
	return friendships, err
}

// Update 更新好友关系
func (r *friendshipRepositoryImpl) Update(ctx context.Context, friendship *model.Friendship) error {
	return r.db.WithContext(ctx).Save(friendship).Error
}

// UpdateRemark 更新备注
func (r *friendshipRepositoryImpl) UpdateRemark(ctx context.Context, userID, friendID, remark string) error {
	return r.db.WithContext(ctx).
		Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", userID, friendID, model.FriendshipStatusNormal).
		Update("remark", remark).Error
}

// Delete 删除好友关系（软删除，更新状态）
func (r *friendshipRepositoryImpl) Delete(ctx context.Context, userID, friendID string) error {
	return r.db.WithContext(ctx).
		Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Update("status", model.FriendshipStatusDeleted).Error
}

// DeleteBidirectional 双向删除好友关系（需在事务中调用）
func (r *friendshipRepositoryImpl) DeleteBidirectional(ctx context.Context, userID, friendID string) error {
	// 删除 A->B
	if err := r.Delete(ctx, userID, friendID); err != nil {
		return err
	}
	// 删除 B->A
	return r.Delete(ctx, friendID, userID)
}

// IsFriend 检查是否是好友
func (r *friendshipRepositoryImpl) IsFriend(ctx context.Context, userID, friendID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", userID, friendID, model.FriendshipStatusNormal).
		Count(&count).Error
	return count > 0, err
}

// WithTx 使用事务
func (r *friendshipRepositoryImpl) WithTx(tx *gorm.DB) FriendshipRepository {
	return &friendshipRepositoryImpl{db: tx}
}
