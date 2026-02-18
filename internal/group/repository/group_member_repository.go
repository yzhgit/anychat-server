package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/group/model"
	"gorm.io/gorm"
)

// GroupMemberRepository 群成员仓库接口
type GroupMemberRepository interface {
	AddMember(ctx context.Context, member *model.GroupMember) error
	AddMembers(ctx context.Context, members []*model.GroupMember) error
	RemoveMember(ctx context.Context, groupID, userID string) error
	UpdateRole(ctx context.Context, groupID, userID, role string) error
	UpdateNickname(ctx context.Context, groupID, userID, nickname string) error
	UpdateMuted(ctx context.Context, groupID, userID string, muted bool) error
	GetMember(ctx context.Context, groupID, userID string) (*model.GroupMember, error)
	GetMembers(ctx context.Context, groupID string, page, pageSize int) ([]*model.GroupMember, int64, error)
	GetMembersByRole(ctx context.Context, groupID, role string) ([]*model.GroupMember, error)
	GetMemberCount(ctx context.Context, groupID string) (int64, error)
	IsMember(ctx context.Context, groupID, userID string) (bool, error)
	GetUserGroups(ctx context.Context, userID string) ([]*model.GroupMember, error)
	GetUserGroupsByUpdateTime(ctx context.Context, userID string, lastUpdateTime time.Time) ([]*model.GroupMember, error)
	WithTx(tx *gorm.DB) GroupMemberRepository
}

// groupMemberRepositoryImpl 群成员仓库实现
type groupMemberRepositoryImpl struct {
	db *gorm.DB
}

// NewGroupMemberRepository 创建群成员仓库
func NewGroupMemberRepository(db *gorm.DB) GroupMemberRepository {
	return &groupMemberRepositoryImpl{db: db}
}

// AddMember 添加成员
func (r *groupMemberRepositoryImpl) AddMember(ctx context.Context, member *model.GroupMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

// AddMembers 批量添加成员
func (r *groupMemberRepositoryImpl) AddMembers(ctx context.Context, members []*model.GroupMember) error {
	return r.db.WithContext(ctx).Create(members).Error
}

// RemoveMember 移除成员
func (r *groupMemberRepositoryImpl) RemoveMember(ctx context.Context, groupID, userID string) error {
	return r.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&model.GroupMember{}).Error
}

// UpdateRole 更新成员角色
func (r *groupMemberRepositoryImpl) UpdateRole(ctx context.Context, groupID, userID, role string) error {
	return r.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Updates(map[string]interface{}{
			"role":       role,
			"updated_at": time.Now(),
		}).Error
}

// UpdateNickname 更新群昵称
func (r *groupMemberRepositoryImpl) UpdateNickname(ctx context.Context, groupID, userID, nickname string) error {
	return r.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Updates(map[string]interface{}{
			"group_nickname": nickname,
			"updated_at":     time.Now(),
		}).Error
}

// UpdateMuted 更新禁言状态
func (r *groupMemberRepositoryImpl) UpdateMuted(ctx context.Context, groupID, userID string, muted bool) error {
	return r.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Updates(map[string]interface{}{
			"is_muted":   muted,
			"updated_at": time.Now(),
		}).Error
}

// GetMember 获取成员信息
func (r *groupMemberRepositoryImpl) GetMember(ctx context.Context, groupID, userID string) (*model.GroupMember, error) {
	var member model.GroupMember
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// GetMembers 获取群成员列表（支持分页）
func (r *groupMemberRepositoryImpl) GetMembers(ctx context.Context, groupID string, page, pageSize int) ([]*model.GroupMember, int64, error) {
	var members []*model.GroupMember
	var total int64

	query := r.db.WithContext(ctx).Model(&model.GroupMember{}).Where("group_id = ?", groupID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Order("joined_at ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&members).Error

	return members, total, err
}

// GetMembersByRole 根据角色获取成员
func (r *groupMemberRepositoryImpl) GetMembersByRole(ctx context.Context, groupID, role string) ([]*model.GroupMember, error) {
	var members []*model.GroupMember
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND role = ?", groupID, role).
		Order("joined_at ASC").
		Find(&members).Error
	return members, err
}

// GetMemberCount 获取成员数量
func (r *groupMemberRepositoryImpl) GetMemberCount(ctx context.Context, groupID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("group_id = ?", groupID).
		Count(&count).Error
	return count, err
}

// IsMember 检查是否是成员
func (r *groupMemberRepositoryImpl) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetUserGroups 获取用户加入的群组列表
func (r *groupMemberRepositoryImpl) GetUserGroups(ctx context.Context, userID string) ([]*model.GroupMember, error) {
	var members []*model.GroupMember
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("joined_at DESC").
		Find(&members).Error
	return members, err
}

// GetUserGroupsByUpdateTime 根据更新时间获取用户群组列表（增量同步）
func (r *groupMemberRepositoryImpl) GetUserGroupsByUpdateTime(ctx context.Context, userID string, lastUpdateTime time.Time) ([]*model.GroupMember, error) {
	var members []*model.GroupMember
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND updated_at > ?", userID, lastUpdateTime).
		Order("updated_at DESC").
		Find(&members).Error
	return members, err
}

// WithTx 使用事务
func (r *groupMemberRepositoryImpl) WithTx(tx *gorm.DB) GroupMemberRepository {
	return &groupMemberRepositoryImpl{db: tx}
}
