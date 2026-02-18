package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/group/model"
	"gorm.io/gorm"
)

// GroupRepository 群组仓库接口
type GroupRepository interface {
	Create(ctx context.Context, group *model.Group) error
	GetByGroupID(ctx context.Context, groupID string) (*model.Group, error)
	Update(ctx context.Context, group *model.Group) error
	UpdateFields(ctx context.Context, groupID string, updates map[string]interface{}) error
	Delete(ctx context.Context, groupID string) error
	UpdateMemberCount(ctx context.Context, groupID string, delta int32) error
	GetGroupsByOwner(ctx context.Context, ownerID string) ([]*model.Group, error)
	Search(ctx context.Context, keyword string, limit int) ([]*model.Group, error)
	WithTx(tx *gorm.DB) GroupRepository
}

// groupRepositoryImpl 群组仓库实现
type groupRepositoryImpl struct {
	db *gorm.DB
}

// NewGroupRepository 创建群组仓库
func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepositoryImpl{db: db}
}

// Create 创建群组
func (r *groupRepositoryImpl) Create(ctx context.Context, group *model.Group) error {
	return r.db.WithContext(ctx).Create(group).Error
}

// GetByGroupID 根据群组ID获取群组
func (r *groupRepositoryImpl) GetByGroupID(ctx context.Context, groupID string) (*model.Group, error) {
	var group model.Group
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND status = ?", groupID, model.GroupStatusNormal).
		First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// Update 更新群组
func (r *groupRepositoryImpl) Update(ctx context.Context, group *model.Group) error {
	return r.db.WithContext(ctx).Save(group).Error
}

// UpdateFields 更新指定字段
func (r *groupRepositoryImpl) UpdateFields(ctx context.Context, groupID string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("group_id = ? AND status = ?", groupID, model.GroupStatusNormal).
		Updates(updates).Error
}

// Delete 删除群组（软删除，更新状态为已解散）
func (r *groupRepositoryImpl) Delete(ctx context.Context, groupID string) error {
	return r.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("group_id = ?", groupID).
		Updates(map[string]interface{}{
			"status":     model.GroupStatusDissolved,
			"updated_at": time.Now(),
		}).Error
}

// UpdateMemberCount 更新成员数量（原子操作）
func (r *groupRepositoryImpl) UpdateMemberCount(ctx context.Context, groupID string, delta int32) error {
	return r.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("group_id = ?", groupID).
		Updates(map[string]interface{}{
			"member_count": gorm.Expr("member_count + ?", delta),
			"updated_at":   time.Now(),
		}).Error
}

// GetGroupsByOwner 获取用户创建的群组
func (r *groupRepositoryImpl) GetGroupsByOwner(ctx context.Context, ownerID string) ([]*model.Group, error) {
	var groups []*model.Group
	err := r.db.WithContext(ctx).
		Where("owner_id = ? AND status = ?", ownerID, model.GroupStatusNormal).
		Order("created_at DESC").
		Find(&groups).Error
	return groups, err
}

// Search 搜索群组（按名称）
func (r *groupRepositoryImpl) Search(ctx context.Context, keyword string, limit int) ([]*model.Group, error) {
	var groups []*model.Group
	err := r.db.WithContext(ctx).
		Where("name LIKE ? AND status = ?", "%"+keyword+"%", model.GroupStatusNormal).
		Order("member_count DESC").
		Limit(limit).
		Find(&groups).Error
	return groups, err
}

// WithTx 使用事务
func (r *groupRepositoryImpl) WithTx(tx *gorm.DB) GroupRepository {
	return &groupRepositoryImpl{db: tx}
}
