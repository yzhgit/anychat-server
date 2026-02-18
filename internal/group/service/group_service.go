package service

import (
	"context"
	"time"

	userpb "github.com/anychat/server/api/proto/user"
	"github.com/anychat/server/internal/group/dto"
	"github.com/anychat/server/internal/group/model"
	"github.com/anychat/server/internal/group/repository"
	"github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GroupService 群组服务接口
type GroupService interface {
	// 群组管理
	CreateGroup(ctx context.Context, ownerID string, req *dto.CreateGroupRequest) (*dto.GroupResponse, error)
	GetGroupInfo(ctx context.Context, userID, groupID string) (*dto.GroupResponse, error)
	UpdateGroup(ctx context.Context, userID, groupID string, req *dto.UpdateGroupRequest) error
	DissolveGroup(ctx context.Context, userID, groupID string) error
	GetUserGroups(ctx context.Context, userID string, lastUpdateTime *int64) (*dto.GroupListResponse, error)

	// 成员管理
	GetGroupMembers(ctx context.Context, userID, groupID string, page, pageSize int) (*dto.GroupMemberListResponse, error)
	InviteMembers(ctx context.Context, userID, groupID string, req *dto.InviteMembersRequest) error
	RemoveMember(ctx context.Context, userID, groupID, targetUserID string) error
	QuitGroup(ctx context.Context, userID, groupID string) error
	UpdateMemberRole(ctx context.Context, userID, groupID, targetUserID string, req *dto.UpdateMemberRoleRequest) error
	UpdateMemberNickname(ctx context.Context, userID, groupID string, req *dto.UpdateMemberNicknameRequest) error
	TransferOwnership(ctx context.Context, userID, groupID, newOwnerID string) error

	// 入群申请
	JoinGroup(ctx context.Context, userID, groupID string, req *dto.JoinGroupRequest) (*dto.JoinGroupResponse, error)
	HandleJoinRequest(ctx context.Context, userID string, requestID int64, req *dto.HandleJoinRequestRequest) error
	GetJoinRequests(ctx context.Context, groupID string, status *string) (*dto.JoinRequestListResponse, error)

	// 群组设置
	UpdateGroupSettings(ctx context.Context, userID, groupID string, req *dto.UpdateGroupSettingsRequest) error
	GetGroupSettings(ctx context.Context, groupID string) (*dto.GroupSettingsResponse, error)

	// 内部gRPC方法（供其他服务调用）
	IsMember(ctx context.Context, groupID, userID string) (bool, string, error)
}

// groupServiceImpl 群组服务实现
type groupServiceImpl struct {
	groupRepo       repository.GroupRepository
	memberRepo      repository.GroupMemberRepository
	settingRepo     repository.GroupSettingRepository
	joinRequestRepo repository.GroupJoinRequestRepository
	userClient      userpb.UserServiceClient
	notificationPub notification.Publisher
	db              *gorm.DB
}

// NewGroupService 创建群组服务
func NewGroupService(
	groupRepo repository.GroupRepository,
	memberRepo repository.GroupMemberRepository,
	settingRepo repository.GroupSettingRepository,
	joinRequestRepo repository.GroupJoinRequestRepository,
	userClient userpb.UserServiceClient,
	notificationPub notification.Publisher,
	db *gorm.DB,
) GroupService {
	return &groupServiceImpl{
		groupRepo:       groupRepo,
		memberRepo:      memberRepo,
		settingRepo:     settingRepo,
		joinRequestRepo: joinRequestRepo,
		userClient:      userClient,
		notificationPub: notificationPub,
		db:              db,
	}
}

// CreateGroup 创建群组
func (s *groupServiceImpl) CreateGroup(ctx context.Context, ownerID string, req *dto.CreateGroupRequest) (*dto.GroupResponse, error) {
	// 验证：至少需要创建者 + 至少1个成员
	if len(req.MemberIDs) == 0 {
		return nil, errors.NewBusiness(errors.CodeGroupMemberTooFew, "至少需要邀请一名成员")
	}

	// 验证：成员数量不能超过最大限制
	totalMembers := len(req.MemberIDs) + 1 // +1 for owner
	if totalMembers > 500 {
		return nil, errors.NewBusiness(errors.CodeGroupMemberLimitReached, "群成员数量超过限制")
	}

	// 生成唯一群组ID
	groupID := uuid.New().String()

	// 创建群组对象
	group := &model.Group{
		GroupID:      groupID,
		Name:         req.Name,
		Avatar:       req.Avatar,
		OwnerID:      ownerID,
		MemberCount:  int32(totalMembers),
		MaxMembers:   500,
		JoinVerify:   req.JoinVerify,
		Status:       model.GroupStatusNormal,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 使用事务创建群组
	err := s.db.Transaction(func(tx *gorm.DB) error {
		groupRepoTx := s.groupRepo.WithTx(tx)
		memberRepoTx := s.memberRepo.WithTx(tx)
		settingRepoTx := s.settingRepo.WithTx(tx)

		// 1. 创建群组记录
		if err := groupRepoTx.Create(ctx, group); err != nil {
			logger.Error("Failed to create group", zap.Error(err))
			return err
		}

		// 2. 创建默认群组设置
		setting := model.DefaultGroupSetting(groupID)
		if err := settingRepoTx.Create(ctx, setting); err != nil {
			logger.Error("Failed to create group settings", zap.Error(err))
			return err
		}

		// 3. 添加群主
		ownerMember := &model.GroupMember{
			GroupID:  groupID,
			UserID:   ownerID,
			Role:     model.GroupRoleOwner,
			JoinedAt: time.Now(),
		}
		if err := memberRepoTx.AddMember(ctx, ownerMember); err != nil {
			logger.Error("Failed to add owner", zap.Error(err))
			return err
		}

		// 4. 添加初始成员
		members := make([]*model.GroupMember, 0, len(req.MemberIDs))
		now := time.Now()
		for _, memberID := range req.MemberIDs {
			if memberID == ownerID {
				continue // 跳过群主
			}
			members = append(members, &model.GroupMember{
				GroupID:  groupID,
				UserID:   memberID,
				Role:     model.GroupRoleMember,
				JoinedAt: now,
			})
		}

		if len(members) > 0 {
			if err := memberRepoTx.AddMembers(ctx, members); err != nil {
				logger.Error("Failed to add members", zap.Error(err))
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 发布成员加入通知（通知所有成员）
	for _, memberID := range req.MemberIDs {
		s.publishMemberJoinedNotification(groupID, memberID, ownerID)
	}

	// 构造响应
	return &dto.GroupResponse{
		GroupID:      group.GroupID,
		Name:         group.Name,
		Avatar:       group.Avatar,
		Announcement: group.Announcement,
		OwnerID:      group.OwnerID,
		MemberCount:  group.MemberCount,
		MaxMembers:   group.MaxMembers,
		JoinVerify:   group.JoinVerify,
		IsMuted:      group.IsMuted,
		MyRole:       model.GroupRoleOwner,
		CreatedAt:    group.CreatedAt,
		UpdatedAt:    group.UpdatedAt,
	}, nil
}

// GetGroupInfo 获取群组信息
func (s *groupServiceImpl) GetGroupInfo(ctx context.Context, userID, groupID string) (*dto.GroupResponse, error) {
	// 获取群组信息
	group, err := s.groupRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeGroupNotFound, "群组不存在")
		}
		logger.Error("Failed to get group", zap.Error(err))
		return nil, err
	}

	// 检查群组状态
	if !group.IsActive() {
		return nil, errors.NewBusiness(errors.CodeGroupDissolved, "群组已解散")
	}

	// 获取用户在群组中的角色
	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	myRole := ""
	if err == nil {
		myRole = member.Role
	}

	return &dto.GroupResponse{
		GroupID:      group.GroupID,
		Name:         group.Name,
		Avatar:       group.Avatar,
		Announcement: group.Announcement,
		OwnerID:      group.OwnerID,
		MemberCount:  group.MemberCount,
		MaxMembers:   group.MaxMembers,
		JoinVerify:   group.JoinVerify,
		IsMuted:      group.IsMuted,
		MyRole:       myRole,
		CreatedAt:    group.CreatedAt,
		UpdatedAt:    group.UpdatedAt,
	}, nil
}

// UpdateGroup 更新群组信息
func (s *groupServiceImpl) UpdateGroup(ctx context.Context, userID, groupID string, req *dto.UpdateGroupRequest) error {
	// 权限检查：只有群主和管理员可以更新
	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}

	if !member.CanManageGroup() {
		return errors.NewBusiness(errors.CodeNoAdminPermission, "无权限更新群信息")
	}

	// 构建更新字段
	updates := make(map[string]any)
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Announcement != nil {
		updates["announcement"] = *req.Announcement
	}
	if req.JoinVerify != nil {
		updates["join_verify"] = *req.JoinVerify
	}

	if len(updates) == 0 {
		return nil // 没有需要更新的字段
	}

	// 更新群组
	if err := s.groupRepo.UpdateFields(ctx, groupID, updates); err != nil {
		logger.Error("Failed to update group", zap.Error(err))
		return err
	}

	// 发布群组信息更新通知
	s.publishGroupInfoUpdatedNotification(groupID, userID, req)

	return nil
}

// DissolveGroup 解散群组
func (s *groupServiceImpl) DissolveGroup(ctx context.Context, userID, groupID string) error {
	// 权限检查：只有群主可以解散
	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}

	if !member.IsOwner() {
		return errors.NewBusiness(errors.CodeNoOwnerPermission, "只有群主可以解散群组")
	}

	// 软删除群组
	if err := s.groupRepo.Delete(ctx, groupID); err != nil {
		logger.Error("Failed to dissolve group", zap.Error(err))
		return err
	}

	// 获取群组信息用于通知
	group, _ := s.groupRepo.GetByGroupID(ctx, groupID)
	groupName := ""
	if group != nil {
		groupName = group.Name
	}

	// 发布群组解散通知
	s.publishGroupDisbandedNotification(groupID, userID, groupName)

	return nil
}

// GetUserGroups 获取用户加入的群组列表
func (s *groupServiceImpl) GetUserGroups(ctx context.Context, userID string, lastUpdateTime *int64) (*dto.GroupListResponse, error) {
	var members []*model.GroupMember
	var err error

	// 增量同步
	if lastUpdateTime != nil && *lastUpdateTime > 0 {
		t := time.Unix(*lastUpdateTime, 0)
		members, err = s.memberRepo.GetUserGroupsByUpdateTime(ctx, userID, t)
	} else {
		members, err = s.memberRepo.GetUserGroups(ctx, userID)
	}

	if err != nil {
		logger.Error("Failed to get user groups", zap.Error(err))
		return nil, err
	}

	// 获取群组详情
	groups := make([]*dto.GroupResponse, 0, len(members))
	for _, member := range members {
		group, err := s.groupRepo.GetByGroupID(ctx, member.GroupID)
		if err != nil {
			logger.Warn("Failed to get group info", zap.String("groupID", member.GroupID), zap.Error(err))
			continue
		}

		if !group.IsActive() {
			continue // 跳过已解散的群组
		}

		groups = append(groups, &dto.GroupResponse{
			GroupID:      group.GroupID,
			Name:         group.Name,
			Avatar:       group.Avatar,
			Announcement: group.Announcement,
			OwnerID:      group.OwnerID,
			MemberCount:  group.MemberCount,
			MaxMembers:   group.MaxMembers,
			JoinVerify:   group.JoinVerify,
			IsMuted:      group.IsMuted,
			MyRole:       member.Role,
			CreatedAt:    group.CreatedAt,
			UpdatedAt:    group.UpdatedAt,
		})
	}

	return &dto.GroupListResponse{
		Groups:     groups,
		Total:      int64(len(groups)),
		UpdateTime: time.Now().Unix(),
	}, nil
}

// GetGroupMembers 获取群成员列表
func (s *groupServiceImpl) GetGroupMembers(ctx context.Context, userID, groupID string, page, pageSize int) (*dto.GroupMemberListResponse, error) {
	// 验证用户是否是群成员
	isMember, err := s.memberRepo.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
	}

	// 默认分页参数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	// 获取成员列表
	members, total, err := s.memberRepo.GetMembers(ctx, groupID, page, pageSize)
	if err != nil {
		logger.Error("Failed to get group members", zap.Error(err))
		return nil, err
	}

	// 转换为DTO
	memberResponses := make([]*dto.GroupMemberResponse, 0, len(members))
	for _, m := range members {
		response := &dto.GroupMemberResponse{
			UserID:   m.UserID,
			Role:     m.Role,
			IsMuted:  m.IsMuted,
			JoinedAt: m.JoinedAt,
		}
		if m.GroupNickname != "" {
			response.GroupNickname = &m.GroupNickname
		}
		memberResponses = append(memberResponses, response)
	}

	return &dto.GroupMemberListResponse{
		Members: memberResponses,
		Total:   total,
		Page:    page,
	}, nil
}

// InviteMembers 邀请成员
func (s *groupServiceImpl) InviteMembers(ctx context.Context, userID, groupID string, req *dto.InviteMembersRequest) error {
	// 获取群组信息
	group, err := s.groupRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeGroupNotFound, "群组不存在")
		}
		return err
	}

	// 检查群组状态
	if !group.IsActive() {
		return errors.NewBusiness(errors.CodeGroupDissolved, "群组已解散")
	}

	// 权限检查：获取邀请人信息
	inviter, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}

	// 检查群设置：普通成员是否可以邀请
	settings, err := s.settingRepo.GetSettings(ctx, groupID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if settings != nil && !settings.AllowMemberInvite && !inviter.CanManageGroup() {
		return errors.NewBusiness(errors.CodeNoAdminPermission, "群设置不允许普通成员邀请")
	}

	// 验证：群成员数量未达上限
	if group.IsFull() {
		return errors.NewBusiness(errors.CodeGroupMemberLimitReached, "群成员已达上限")
	}

	// 处理每个被邀请人
	for _, inviteeID := range req.UserIDs {
		// 检查是否已经是成员
		isMember, _ := s.memberRepo.IsMember(ctx, groupID, inviteeID)
		if isMember {
			continue // 跳过已经是成员的用户
		}

		// 如果需要验证，创建入群申请
		if group.JoinVerify {
			request := &model.GroupJoinRequest{
				GroupID:   groupID,
				UserID:    inviteeID,
				InviterID: userID,
				Status:    model.JoinRequestStatusPending,
				CreatedAt: time.Now(),
			}
			if err := s.joinRequestRepo.Create(ctx, request); err != nil {
				logger.Error("Failed to create join request", zap.Error(err))
				continue
			}
		} else {
			// 直接添加成员
			err := s.db.Transaction(func(tx *gorm.DB) error {
				memberRepoTx := s.memberRepo.WithTx(tx)
				groupRepoTx := s.groupRepo.WithTx(tx)

				member := &model.GroupMember{
					GroupID:  groupID,
					UserID:   inviteeID,
					Role:     model.GroupRoleMember,
					JoinedAt: time.Now(),
				}
				if err := memberRepoTx.AddMember(ctx, member); err != nil {
					return err
				}

				// 更新成员数量
				return groupRepoTx.UpdateMemberCount(ctx, groupID, 1)
			})

			if err != nil {
				logger.Error("Failed to add member", zap.String("inviteeID", inviteeID), zap.Error(err))
				continue
			}
		}
	}

	return nil
}

// RemoveMember 移除成员
func (s *groupServiceImpl) RemoveMember(ctx context.Context, userID, groupID, targetUserID string) error {
	// 获取操作者信息
	operator, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}

	// 获取目标成员信息
	target, err := s.memberRepo.GetMember(ctx, groupID, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "目标用户不是群成员")
		}
		return err
	}

	// 权限检查
	if target.IsOwner() {
		return errors.NewBusiness(errors.CodeCannotRemoveOwner, "不能移除群主")
	}

	if !operator.CanRemoveMember(target.Role) {
		if target.Role == model.GroupRoleAdmin {
			return errors.NewBusiness(errors.CodeCannotRemoveAdmin, "管理员不能移除其他管理员")
		}
		return errors.NewBusiness(errors.CodeNoAdminPermission, "无权限移除成员")
	}

	// 使用事务删除成员并更新计数
	err = s.db.Transaction(func(tx *gorm.DB) error {
		memberRepoTx := s.memberRepo.WithTx(tx)
		groupRepoTx := s.groupRepo.WithTx(tx)

		if err := memberRepoTx.RemoveMember(ctx, groupID, targetUserID); err != nil {
			return err
		}

		return groupRepoTx.UpdateMemberCount(ctx, groupID, -1)
	})

	if err != nil {
		logger.Error("Failed to remove member", zap.Error(err))
		return err
	}

	// 发布成员离开通知
	s.publishMemberLeftNotification(groupID, targetUserID, userID, "removed_by_admin")

	return nil
}

// QuitGroup 退出群组
func (s *groupServiceImpl) QuitGroup(ctx context.Context, userID, groupID string) error {
	// 获取成员信息
	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}

	// 群主不能直接退出
	if member.IsOwner() {
		// 检查是否只剩群主一人
		count, err := s.memberRepo.GetMemberCount(ctx, groupID)
		if err != nil {
			return err
		}

		if count == 1 {
			// 自动解散群组
			return s.DissolveGroup(ctx, userID, groupID)
		}

		return errors.NewBusiness(errors.CodeCannotQuitOwnGroup, "群主不能退出群组，请先转让群主或解散群组")
	}

	// 使用事务删除成员并更新计数
	err = s.db.Transaction(func(tx *gorm.DB) error {
		memberRepoTx := s.memberRepo.WithTx(tx)
		groupRepoTx := s.groupRepo.WithTx(tx)

		if err := memberRepoTx.RemoveMember(ctx, groupID, userID); err != nil {
			return err
		}

		return groupRepoTx.UpdateMemberCount(ctx, groupID, -1)
	})

	if err != nil {
		logger.Error("Failed to quit group", zap.Error(err))
		return err
	}

	// 发布成员离开通知
	s.publishMemberLeftNotification(groupID, userID, userID, "self_quit")

	return nil
}

// UpdateMemberRole 更新成员角色
func (s *groupServiceImpl) UpdateMemberRole(ctx context.Context, userID, groupID, targetUserID string, req *dto.UpdateMemberRoleRequest) error {
	// 权限检查：只有群主可以设置角色
	operator, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}

	if !operator.IsOwner() {
		return errors.NewBusiness(errors.CodeNoOwnerPermission, "只有群主可以设置管理员")
	}

	// 验证目标成员存在
	target, err := s.memberRepo.GetMember(ctx, groupID, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "目标用户不是群成员")
		}
		return err
	}

	// 不能修改群主角色
	if target.IsOwner() {
		return errors.NewBusiness(errors.CodeNoOwnerPermission, "不能修改群主角色")
	}

	// 更新角色
	if err := s.memberRepo.UpdateRole(ctx, groupID, targetUserID, req.Role); err != nil {
		logger.Error("Failed to update member role", zap.Error(err))
		return err
	}

	// 发布角色变更通知
	s.publishRoleChangedNotification(groupID, targetUserID, target.Role, req.Role, userID)

	return nil
}

// UpdateMemberNickname 更新群昵称
func (s *groupServiceImpl) UpdateMemberNickname(ctx context.Context, userID, groupID string, req *dto.UpdateMemberNicknameRequest) error {
	// 验证用户是否是群成员
	isMember, err := s.memberRepo.IsMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
	}

	// 更新昵称
	if err := s.memberRepo.UpdateNickname(ctx, groupID, userID, req.Nickname); err != nil {
		logger.Error("Failed to update member nickname", zap.Error(err))
		return err
	}

	return nil
}

// TransferOwnership 转让群主
func (s *groupServiceImpl) TransferOwnership(ctx context.Context, userID, groupID, newOwnerID string) error {
	// 权限检查：只有群主可以转让
	operator, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}

	if !operator.IsOwner() {
		return errors.NewBusiness(errors.CodeNoOwnerPermission, "只有群主可以转让群组")
	}

	// 验证新群主是群成员
	newOwner, err := s.memberRepo.GetMember(ctx, groupID, newOwnerID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "新群主不是群成员")
		}
		return err
	}

	if newOwner.IsOwner() {
		return nil // 已经是群主，无需操作
	}

	// 使用事务转让群主
	err = s.db.Transaction(func(tx *gorm.DB) error {
		memberRepoTx := s.memberRepo.WithTx(tx)
		groupRepoTx := s.groupRepo.WithTx(tx)

		// 1. 更新群组owner_id
		if err := groupRepoTx.UpdateFields(ctx, groupID, map[string]any{"owner_id": newOwnerID}); err != nil {
			return err
		}

		// 2. 将原群主改为管理员
		if err := memberRepoTx.UpdateRole(ctx, groupID, userID, model.GroupRoleAdmin); err != nil {
			return err
		}

		// 3. 将新群主改为owner
		return memberRepoTx.UpdateRole(ctx, groupID, newOwnerID, model.GroupRoleOwner)
	})

	if err != nil {
		logger.Error("Failed to transfer ownership", zap.Error(err))
		return err
	}

	return nil
}

// JoinGroup 加入群组
func (s *groupServiceImpl) JoinGroup(ctx context.Context, userID, groupID string, req *dto.JoinGroupRequest) (*dto.JoinGroupResponse, error) {
	// 获取群组信息
	group, err := s.groupRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeGroupNotFound, "群组不存在")
		}
		return nil, err
	}

	// 检查群组状态
	if !group.IsActive() {
		return nil, errors.NewBusiness(errors.CodeGroupDissolved, "群组已解散")
	}

	// 检查是否已经是成员
	isMember, err := s.memberRepo.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if isMember {
		return nil, errors.NewBusiness(errors.CodeAlreadyGroupMember, "你已经是群成员")
	}

	// 检查是否已经有待处理的申请
	existingRequest, err := s.joinRequestRepo.GetExistingRequest(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if existingRequest != nil {
		return &dto.JoinGroupResponse{
			NeedVerify: true,
			RequestID:  &existingRequest.ID,
			Message:    "你已经有待处理的申请",
		}, nil
	}

	// 检查是否需要验证
	if group.JoinVerify {
		// 创建入群申请
		request := &model.GroupJoinRequest{
			GroupID:   groupID,
			UserID:    userID,
			Message:   req.Message,
			Status:    model.JoinRequestStatusPending,
			CreatedAt: time.Now(),
		}
		if err := s.joinRequestRepo.Create(ctx, request); err != nil {
			logger.Error("Failed to create join request", zap.Error(err))
			return nil, err
		}

		return &dto.JoinGroupResponse{
			NeedVerify: true,
			RequestID:  &request.ID,
			Message:    "申请已提交，等待审核",
		}, nil
	}

	// 直接加入群组
	err = s.db.Transaction(func(tx *gorm.DB) error {
		memberRepoTx := s.memberRepo.WithTx(tx)
		groupRepoTx := s.groupRepo.WithTx(tx)

		member := &model.GroupMember{
			GroupID:  groupID,
			UserID:   userID,
			Role:     model.GroupRoleMember,
			JoinedAt: time.Now(),
		}
		if err := memberRepoTx.AddMember(ctx, member); err != nil {
			return err
		}

		return groupRepoTx.UpdateMemberCount(ctx, groupID, 1)
	})

	if err != nil {
		logger.Error("Failed to join group", zap.Error(err))
		return nil, err
	}

	return &dto.JoinGroupResponse{
		NeedVerify: false,
		Message:    "成功加入群组",
	}, nil
}

// HandleJoinRequest 处理入群申请
func (s *groupServiceImpl) HandleJoinRequest(ctx context.Context, userID string, requestID int64, req *dto.HandleJoinRequestRequest) error {
	// 获取申请信息
	request, err := s.joinRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeJoinRequestNotFound, "入群申请不存在")
		}
		return err
	}

	// 检查申请状态
	if request.IsProcessed() {
		return errors.NewBusiness(errors.CodeJoinRequestProcessed, "申请已处理")
	}

	// 权限检查：群主和管理员可以处理
	operator, err := s.memberRepo.GetMember(ctx, request.GroupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}

	if !operator.CanManageGroup() {
		return errors.NewBusiness(errors.CodeNoAdminPermission, "无权限处理申请")
	}

	// 处理申请
	if req.Accept {
		// 接受申请：使用事务
		err = s.db.Transaction(func(tx *gorm.DB) error {
			joinRequestRepoTx := s.joinRequestRepo.WithTx(tx)
			memberRepoTx := s.memberRepo.WithTx(tx)
			groupRepoTx := s.groupRepo.WithTx(tx)

			// 1. 更新申请状态
			if err := joinRequestRepoTx.UpdateStatus(ctx, requestID, model.JoinRequestStatusAccepted); err != nil {
				return err
			}

			// 2. 添加成员
			member := &model.GroupMember{
				GroupID:  request.GroupID,
				UserID:   request.UserID,
				Role:     model.GroupRoleMember,
				JoinedAt: time.Now(),
			}
			if err := memberRepoTx.AddMember(ctx, member); err != nil {
				return err
			}

			// 3. 更新成员数量
			return groupRepoTx.UpdateMemberCount(ctx, request.GroupID, 1)
		})

		if err != nil {
			logger.Error("Failed to accept join request", zap.Error(err))
			return err
		}
	} else {
		// 拒绝申请
		if err := s.joinRequestRepo.UpdateStatus(ctx, requestID, model.JoinRequestStatusRejected); err != nil {
			logger.Error("Failed to reject join request", zap.Error(err))
			return err
		}
	}

	return nil
}

// GetJoinRequests 获取入群申请列表
func (s *groupServiceImpl) GetJoinRequests(ctx context.Context, groupID string, status *string) (*dto.JoinRequestListResponse, error) {
	// 获取申请列表
	requests, err := s.joinRequestRepo.GetRequestsByGroup(ctx, groupID, status)
	if err != nil {
		logger.Error("Failed to get join requests", zap.Error(err))
		return nil, err
	}

	// 转换为DTO
	requestResponses := make([]*dto.JoinRequestResponse, 0, len(requests))
	for _, r := range requests {
		response := &dto.JoinRequestResponse{
			ID:        r.ID,
			GroupID:   r.GroupID,
			UserID:    r.UserID,
			Message:   r.Message,
			Status:    r.Status,
			CreatedAt: r.CreatedAt,
		}
		if r.InviterID != "" {
			response.InviterID = &r.InviterID
		}
		requestResponses = append(requestResponses, response)
	}

	return &dto.JoinRequestListResponse{
		Requests: requestResponses,
		Total:    int64(len(requestResponses)),
	}, nil
}

// UpdateGroupSettings 更新群组设置
func (s *groupServiceImpl) UpdateGroupSettings(ctx context.Context, userID, groupID string, req *dto.UpdateGroupSettingsRequest) error {
	// 权限检查：只有群主和管理员可以更新设置
	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}

	if !member.CanManageGroup() {
		return errors.NewBusiness(errors.CodeNoAdminPermission, "无权限更新群设置")
	}

	// 构建更新字段
	updates := make(map[string]any)
	if req.AllowMemberInvite != nil {
		updates["allow_member_invite"] = *req.AllowMemberInvite
	}
	if req.AllowViewHistory != nil {
		updates["allow_view_history"] = *req.AllowViewHistory
	}
	if req.AllowAddFriend != nil {
		updates["allow_add_friend"] = *req.AllowAddFriend
	}
	if req.ShowMemberNickname != nil {
		updates["show_member_nickname"] = *req.ShowMemberNickname
	}

	if len(updates) == 0 {
		return nil
	}

	// 更新设置
	if err := s.settingRepo.UpdateSettings(ctx, groupID, updates); err != nil {
		logger.Error("Failed to update group settings", zap.Error(err))
		return err
	}

	return nil
}

// GetGroupSettings 获取群组设置
func (s *groupServiceImpl) GetGroupSettings(ctx context.Context, groupID string) (*dto.GroupSettingsResponse, error) {
	settings, err := s.settingRepo.GetSettings(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回默认设置
			return &dto.GroupSettingsResponse{
				GroupID:            groupID,
				AllowMemberInvite:  true,
				AllowViewHistory:   true,
				AllowAddFriend:     true,
				ShowMemberNickname: true,
			}, nil
		}
		logger.Error("Failed to get group settings", zap.Error(err))
		return nil, err
	}

	return &dto.GroupSettingsResponse{
		GroupID:            settings.GroupID,
		AllowMemberInvite:  settings.AllowMemberInvite,
		AllowViewHistory:   settings.AllowViewHistory,
		AllowAddFriend:     settings.AllowAddFriend,
		ShowMemberNickname: settings.ShowMemberNickname,
	}, nil
}

// IsMember 检查是否为群成员（供其他服务调用）
func (s *groupServiceImpl) IsMember(ctx context.Context, groupID, userID string) (bool, string, error) {
	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, "", nil
		}
		return false, "", err
	}

	return true, member.Role, nil
}

// publishMemberJoinedNotification 发布成员加入通知
func (s *groupServiceImpl) publishMemberJoinedNotification(groupID, userID, inviterID string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"group_id":       groupID,
		"user_id":        userID,
		"inviter_user_id": inviterID,
		"joined_at":      time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeGroupMemberJoined,
		inviterID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToGroup(groupID, notif); err != nil {
		logger.Error("Failed to publish member joined notification",
			zap.String("groupId", groupID),
			zap.Error(err))
	}
}

// publishMemberLeftNotification 发布成员离开通知
func (s *groupServiceImpl) publishMemberLeftNotification(groupID, userID, operatorID, reason string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"group_id":         groupID,
		"user_id":          userID,
		"operator_user_id": operatorID,
		"reason":           reason,
		"left_at":          time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeGroupMemberLeft,
		operatorID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToGroup(groupID, notif); err != nil {
		logger.Error("Failed to publish member left notification",
			zap.String("groupId", groupID),
			zap.Error(err))
	}
}

// publishGroupInfoUpdatedNotification 发布群组信息更新通知
func (s *groupServiceImpl) publishGroupInfoUpdatedNotification(groupID, operatorID string, req *dto.UpdateGroupRequest) {
	if s.notificationPub == nil {
		return
	}

	updatedFields := []string{}
	payload := map[string]interface{}{
		"group_id":         groupID,
		"operator_user_id": operatorID,
		"updated_at":       time.Now().Unix(),
	}

	if req.Name != nil {
		updatedFields = append(updatedFields, "name")
		payload["group_name"] = *req.Name
	}
	if req.Avatar != nil {
		updatedFields = append(updatedFields, "avatar")
		payload["group_avatar"] = *req.Avatar
	}
	if req.Announcement != nil {
		updatedFields = append(updatedFields, "announcement")
		payload["announcement"] = *req.Announcement
	}

	payload["updated_fields"] = updatedFields

	notif := notification.NewNotification(
		notification.TypeGroupInfoUpdated,
		operatorID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToGroup(groupID, notif); err != nil {
		logger.Error("Failed to publish group info updated notification",
			zap.String("groupId", groupID),
			zap.Error(err))
	}
}

// publishRoleChangedNotification 发布角色变更通知
func (s *groupServiceImpl) publishRoleChangedNotification(groupID, userID, oldRole, newRole, operatorID string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"group_id":         groupID,
		"user_id":          userID,
		"old_role":         oldRole,
		"new_role":         newRole,
		"operator_user_id": operatorID,
		"changed_at":       time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeGroupRoleChanged,
		operatorID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToGroup(groupID, notif); err != nil {
		logger.Error("Failed to publish role changed notification",
			zap.String("groupId", groupID),
			zap.Error(err))
	}
}

// publishGroupDisbandedNotification 发布群组解散通知
func (s *groupServiceImpl) publishGroupDisbandedNotification(groupID, operatorID, groupName string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"group_id":         groupID,
		"group_name":       groupName,
		"operator_user_id": operatorID,
		"disbanded_at":     time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeGroupDisbanded,
		operatorID,
		notification.PriorityHigh,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToGroup(groupID, notif); err != nil {
		logger.Error("Failed to publish group disbanded notification",
			zap.String("groupId", groupID),
			zap.Error(err))
	}
}
