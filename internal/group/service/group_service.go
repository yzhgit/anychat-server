package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	messagepb "github.com/anychat/server/api/proto/message"
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
	MuteMember(ctx context.Context, userID, groupID, targetUserID string, req *dto.MuteMemberRequest) error
	UnmuteMember(ctx context.Context, userID, groupID, targetUserID string) error

	// 入群申请
	JoinGroup(ctx context.Context, userID, groupID string, req *dto.JoinGroupRequest) (*dto.JoinGroupResponse, error)
	HandleJoinRequest(ctx context.Context, userID string, requestID int64, req *dto.HandleJoinRequestRequest) error
	GetJoinRequests(ctx context.Context, userID, groupID string, status *string) (*dto.JoinRequestListResponse, error)
	PinGroupMessage(ctx context.Context, userID, groupID, messageID string) error
	UnpinGroupMessage(ctx context.Context, userID, groupID, messageID string) error
	GetPinnedMessages(ctx context.Context, userID, groupID string) (*dto.PinnedMessageListResponse, error)
	SetGroupMute(ctx context.Context, userID, groupID string, enabled bool) error

	// 群组设置
	UpdateGroupSettings(ctx context.Context, userID, groupID string, req *dto.UpdateGroupSettingsRequest) error
	GetGroupSettings(ctx context.Context, groupID string) (*dto.GroupSettingsResponse, error)

	// 群备注
	UpdateMemberRemark(ctx context.Context, userID, groupID string, req *dto.UpdateMemberRemarkRequest) error

	// 群二维码
	GetGroupQRCode(ctx context.Context, userID, groupID string) (*dto.GroupQRCodeResponse, error)
	RefreshGroupQRCode(ctx context.Context, userID, groupID string) (*dto.GroupQRCodeResponse, error)
	JoinGroupByQRCode(ctx context.Context, userID, token string) (*dto.JoinGroupByQRCodeResponse, error)

	// 内部gRPC方法（供其他服务调用）
	IsMember(ctx context.Context, groupID, userID string) (bool, string, error)
}

// groupServiceImpl 群组服务实现
type groupServiceImpl struct {
	groupRepo       repository.GroupRepository
	memberRepo      repository.GroupMemberRepository
	settingRepo     repository.GroupSettingRepository
	joinRequestRepo repository.GroupJoinRequestRepository
	pinnedRepo      repository.GroupPinnedMessageRepository
	qrcodeRepo      repository.GroupQRCodeRepository
	messageClient   messagepb.MessageServiceClient
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
	pinnedRepo repository.GroupPinnedMessageRepository,
	qrcodeRepo repository.GroupQRCodeRepository,
	messageClient messagepb.MessageServiceClient,
	userClient userpb.UserServiceClient,
	notificationPub notification.Publisher,
	db *gorm.DB,
) GroupService {
	return &groupServiceImpl{
		groupRepo:       groupRepo,
		memberRepo:      memberRepo,
		settingRepo:     settingRepo,
		joinRequestRepo: joinRequestRepo,
		pinnedRepo:      pinnedRepo,
		qrcodeRepo:      qrcodeRepo,
		messageClient:   messageClient,
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
		GroupID:     groupID,
		Name:        req.Name,
		Avatar:      req.Avatar,
		OwnerID:     ownerID,
		MemberCount: int32(totalMembers),
		MaxMembers:  500,
		Status:      model.GroupStatusNormal,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
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
		Description:  group.Description,
		OwnerID:      group.OwnerID,
		MemberCount:  group.MemberCount,
		MaxMembers:   group.MaxMembers,
		JoinVerify:   true,
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
	displayName := group.Name
	if err == nil {
		myRole = member.Role
		if member.GroupRemark != "" {
			displayName = member.GroupRemark
		}
	}
	joinVerify := s.getGroupJoinVerify(ctx, groupID)

	return &dto.GroupResponse{
		GroupID:      group.GroupID,
		Name:         group.Name,
		DisplayName:  displayName,
		Avatar:       group.Avatar,
		Announcement: group.Announcement,
		Description:  group.Description,
		OwnerID:      group.OwnerID,
		MemberCount:  group.MemberCount,
		MaxMembers:   group.MaxMembers,
		JoinVerify:   joinVerify,
		IsMuted:      group.IsMuted,
		MyRole:       myRole,
		CreatedAt:    group.CreatedAt,
		UpdatedAt:    group.UpdatedAt,
	}, nil
}

// UpdateGroup 更新群组信息
func (s *groupServiceImpl) UpdateGroup(ctx context.Context, userID, groupID string, req *dto.UpdateGroupRequest) error {
	// 权限检查：群主和管理员可修改；普通成员需群设置允许
	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}

	if !member.CanManageGroup() {
		settings, settingsErr := s.settingRepo.GetSettings(ctx, groupID)
		if settingsErr != nil && settingsErr != gorm.ErrRecordNotFound {
			return settingsErr
		}
		if settings == nil || !settings.AllowMemberModify {
			return errors.NewBusiness(errors.CodeNoAdminPermission, "无权限更新群信息")
		}
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
	if req.Description != nil {
		updates["description"] = *req.Description
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

		displayName := group.Name
		if member.GroupRemark != "" {
			displayName = member.GroupRemark
		}

		groups = append(groups, &dto.GroupResponse{
			GroupID:      group.GroupID,
			Name:         group.Name,
			DisplayName:  displayName,
			Avatar:       group.Avatar,
			Announcement: group.Announcement,
			Description:  group.Description,
			OwnerID:      group.OwnerID,
			MemberCount:  group.MemberCount,
			MaxMembers:   group.MaxMembers,
			JoinVerify:   s.getGroupJoinVerify(ctx, group.GroupID),
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
			UserID:     m.UserID,
			Role:       m.Role,
			IsMuted:    m.IsMutedNow(),
			MutedUntil: m.MutedUntil,
			JoinedAt:   m.JoinedAt,
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
	joinVerify := true
	if settings != nil {
		joinVerify = settings.JoinVerify
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
		if joinVerify {
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

// MuteMember 禁言成员
func (s *groupServiceImpl) MuteMember(ctx context.Context, userID, groupID, targetUserID string, req *dto.MuteMemberRequest) error {
	operator, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}
	target, err := s.memberRepo.GetMember(ctx, groupID, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "目标用户不是群成员")
		}
		return err
	}

	if target.IsOwner() {
		return errors.NewBusiness(errors.CodeNoAdminPermission, "不能禁言群主")
	}
	if !operator.CanMuteMember(target.Role) {
		return errors.NewBusiness(errors.CodeNoAdminPermission, "无权限禁言成员")
	}

	var mutedUntil *time.Time
	switch req.Type {
	case "permanent":
		permanent := model.PermanentMutedUntil
		mutedUntil = &permanent
	case "temporary":
		if req.DurationMinutes <= 0 {
			return errors.NewBusiness(errors.CodeParamError, "临时禁言时长必须大于0")
		}
		t := time.Now().Add(time.Duration(req.DurationMinutes) * time.Minute)
		mutedUntil = &t
	default:
		return errors.NewBusiness(errors.CodeParamError, "禁言类型无效")
	}

	if err := s.memberRepo.UpdateMutedUntil(ctx, groupID, targetUserID, mutedUntil); err != nil {
		logger.Error("Failed to mute member", zap.Error(err))
		return err
	}

	s.publishMemberMutedNotification(groupID, userID, targetUserID, mutedUntil)
	return nil
}

// UnmuteMember 解除禁言
func (s *groupServiceImpl) UnmuteMember(ctx context.Context, userID, groupID, targetUserID string) error {
	operator, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}
	target, err := s.memberRepo.GetMember(ctx, groupID, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "目标用户不是群成员")
		}
		return err
	}
	if target.IsOwner() {
		return errors.NewBusiness(errors.CodeNoAdminPermission, "不能操作群主")
	}
	if !operator.CanMuteMember(target.Role) {
		return errors.NewBusiness(errors.CodeNoAdminPermission, "无权限解除禁言")
	}
	if err := s.memberRepo.UpdateMutedUntil(ctx, groupID, targetUserID, nil); err != nil {
		logger.Error("Failed to unmute member", zap.Error(err))
		return err
	}
	s.publishMemberUnmutedNotification(groupID, userID, targetUserID)
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
	joinVerify := s.getGroupJoinVerify(ctx, groupID)
	if joinVerify {
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
func (s *groupServiceImpl) GetJoinRequests(ctx context.Context, userID, groupID string, status *string) (*dto.JoinRequestListResponse, error) {
	operator, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return nil, err
	}
	if !operator.CanManageGroup() {
		return nil, errors.NewBusiness(errors.CodeNoAdminPermission, "无权限查看入群申请")
	}

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

// PinGroupMessage 置顶群消息
func (s *groupServiceImpl) PinGroupMessage(ctx context.Context, userID, groupID, messageID string) error {
	group, err := s.groupRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeGroupNotFound, "群组不存在")
		}
		return err
	}
	if !group.IsActive() {
		return errors.NewBusiness(errors.CodeGroupDissolved, "群组已解散")
	}

	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}
	if !member.CanManageGroup() {
		return errors.NewBusiness(errors.CodeNoAdminPermission, "无权限置顶消息")
	}

	content, err := s.getPinnedMessageContent(ctx, groupID, messageID)
	if err != nil {
		return err
	}

	record := &model.GroupPinnedMessage{
		GroupID:   groupID,
		MessageID: messageID,
		PinnedBy:  userID,
		Content:   content,
		CreatedAt: time.Now(),
	}
	if err := s.pinnedRepo.Upsert(ctx, record); err != nil {
		logger.Error("Failed to pin group message", zap.Error(err))
		return err
	}

	s.publishGroupMessagePinnedNotification(groupID, userID, messageID)
	return nil
}

// UnpinGroupMessage 取消置顶群消息
func (s *groupServiceImpl) UnpinGroupMessage(ctx context.Context, userID, groupID, messageID string) error {
	group, err := s.groupRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeGroupNotFound, "群组不存在")
		}
		return err
	}
	if !group.IsActive() {
		return errors.NewBusiness(errors.CodeGroupDissolved, "群组已解散")
	}

	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}
	if !member.CanManageGroup() {
		return errors.NewBusiness(errors.CodeNoAdminPermission, "无权限取消置顶")
	}

	if err := s.pinnedRepo.Delete(ctx, groupID, messageID); err != nil {
		logger.Error("Failed to unpin group message", zap.Error(err))
		return err
	}

	s.publishGroupMessageUnpinnedNotification(groupID, userID, messageID)
	return nil
}

// GetPinnedMessages 获取群置顶消息
func (s *groupServiceImpl) GetPinnedMessages(ctx context.Context, userID, groupID string) (*dto.PinnedMessageListResponse, error) {
	group, err := s.groupRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeGroupNotFound, "群组不存在")
		}
		return nil, err
	}
	if !group.IsActive() {
		return nil, errors.NewBusiness(errors.CodeGroupDissolved, "群组已解散")
	}

	isMember, err := s.memberRepo.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
	}

	records, err := s.pinnedRepo.ListByGroup(ctx, groupID)
	if err != nil {
		logger.Error("Failed to list pinned messages", zap.Error(err))
		return nil, err
	}

	resp := &dto.PinnedMessageListResponse{
		Messages: make([]*dto.PinnedMessageResponse, 0, len(records)),
	}
	for _, item := range records {
		resp.Messages = append(resp.Messages, &dto.PinnedMessageResponse{
			MessageID: item.MessageID,
			Content:   item.Content,
			PinnedBy:  item.PinnedBy,
			PinnedAt:  item.CreatedAt.Unix(),
		})
	}
	return resp, nil
}

// SetGroupMute 设置全体禁言
func (s *groupServiceImpl) SetGroupMute(ctx context.Context, userID, groupID string, enabled bool) error {
	group, err := s.groupRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeGroupNotFound, "群组不存在")
		}
		return err
	}
	if !group.IsActive() {
		return errors.NewBusiness(errors.CodeGroupDissolved, "群组已解散")
	}

	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return err
	}
	if !member.CanManageGroup() {
		return errors.NewBusiness(errors.CodeNoAdminPermission, "无权限设置全体禁言")
	}

	if err := s.groupRepo.UpdateFields(ctx, groupID, map[string]any{"is_muted": enabled}); err != nil {
		logger.Error("Failed to set group mute", zap.Error(err))
		return err
	}

	messageText := "已关闭全体禁言"
	if enabled {
		messageText = "已开启全体禁言"
	}
	if err := s.sendGroupSystemMessage(ctx, groupID, userID, messageText); err != nil {
		logger.Warn("Failed to send group mute system message", zap.String("groupID", groupID), zap.Error(err))
	}

	s.publishGroupMutedNotification(groupID, userID, enabled)
	return nil
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
	if req.JoinVerify != nil {
		updates["join_verify"] = *req.JoinVerify
	}
	if req.AllowMemberInvite != nil {
		updates["allow_member_invite"] = *req.AllowMemberInvite
	}
	if req.AllowViewHistory != nil {
		updates["allow_view_history"] = *req.AllowViewHistory
	}
	if req.AllowAddFriend != nil {
		updates["allow_add_friend"] = *req.AllowAddFriend
	}
	if req.AllowMemberModify != nil {
		updates["allow_member_modify"] = *req.AllowMemberModify
	}

	if len(updates) == 0 {
		return nil
	}

	// 更新设置
	if err := s.settingRepo.UpdateSettings(ctx, groupID, updates); err != nil {
		logger.Error("Failed to update group settings", zap.Error(err))
		return err
	}

	s.publishGroupSettingsUpdatedNotification(groupID, userID, updates)
	return nil
}

// GetGroupSettings 获取群组设置
func (s *groupServiceImpl) GetGroupSettings(ctx context.Context, groupID string) (*dto.GroupSettingsResponse, error) {
	settings, err := s.settingRepo.GetSettings(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回默认设置
			return &dto.GroupSettingsResponse{
				GroupID:           groupID,
				JoinVerify:        true,
				AllowMemberInvite: true,
				AllowViewHistory:  true,
				AllowAddFriend:    true,
				AllowMemberModify: false,
			}, nil
		}
		logger.Error("Failed to get group settings", zap.Error(err))
		return nil, err
	}

	return &dto.GroupSettingsResponse{
		GroupID:           settings.GroupID,
		JoinVerify:        settings.JoinVerify,
		AllowMemberInvite: settings.AllowMemberInvite,
		AllowViewHistory:  settings.AllowViewHistory,
		AllowAddFriend:    settings.AllowAddFriend,
		AllowMemberModify: settings.AllowMemberModify,
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

// UpdateMemberRemark 设置/清空群备注（仅对操作者自己可见）
func (s *groupServiceImpl) UpdateMemberRemark(ctx context.Context, userID, groupID string, req *dto.UpdateMemberRemarkRequest) error {
	isMember, err := s.memberRepo.IsMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
	}

	if err := s.memberRepo.UpdateRemark(ctx, groupID, userID, req.Remark); err != nil {
		logger.Error("Failed to update member remark", zap.Error(err))
		return err
	}
	return nil
}

// GetGroupQRCode 获取群二维码（有效则返回，快过期则自动续期，无则创建）
func (s *groupServiceImpl) GetGroupQRCode(ctx context.Context, userID, groupID string) (*dto.GroupQRCodeResponse, error) {
	isMember, err := s.memberRepo.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
	}

	qr, err := s.qrcodeRepo.GetActiveByGroupID(ctx, groupID)
	if err == nil && qr.IsValid() {
		// 距过期不足 1 天时自动续期
		if time.Until(qr.ExpireAt) < model.QRCodeRenewThreshold {
			newExpire := time.Now().Add(model.DefaultQRCodeTTL)
			if renewErr := s.qrcodeRepo.UpdateExpireAt(ctx, qr.Token, newExpire); renewErr != nil {
				logger.Warn("Failed to renew qrcode", zap.Error(renewErr))
			} else {
				qr.ExpireAt = newExpire
			}
		}
		return buildQRCodeResponse(qr), nil
	}

	// 创建新二维码
	return s.createNewQRCode(ctx, userID, groupID)
}

// RefreshGroupQRCode 刷新群二维码（使旧码立即失效��仅群主/管理员）
func (s *groupServiceImpl) RefreshGroupQRCode(ctx context.Context, userID, groupID string) (*dto.GroupQRCodeResponse, error) {
	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeNotGroupMember, "你不是群成员")
		}
		return nil, err
	}
	if !member.CanManageGroup() {
		return nil, errors.NewBusiness(errors.CodeNoAdminPermission, "仅群主或管理员可刷新二维码")
	}

	// 失效旧码
	if err := s.qrcodeRepo.InvalidateByGroupID(ctx, groupID); err != nil {
		logger.Error("Failed to invalidate old qrcode", zap.Error(err))
		return nil, err
	}

	return s.createNewQRCode(ctx, userID, groupID)
}

// JoinGroupByQRCode 扫码加入群组
func (s *groupServiceImpl) JoinGroupByQRCode(ctx context.Context, userID, token string) (*dto.JoinGroupByQRCodeResponse, error) {
	qr, err := s.qrcodeRepo.GetByToken(ctx, token)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeGroupQRInvalid, "二维码无效")
		}
		return nil, err
	}

	if !qr.IsActive {
		return nil, errors.NewBusiness(errors.CodeGroupQRInvalid, "二维码已失效，请重新获取")
	}
	if !qr.IsValid() {
		return nil, errors.NewBusiness(errors.CodeGroupQRExpired, "二维码已过期，请重新获取")
	}

	groupID := qr.GroupID

	group, err := s.groupRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeGroupNotFound, "群组不存在")
		}
		return nil, err
	}
	if !group.IsActive() {
		return nil, errors.NewBusiness(errors.CodeGroupDissolved, "群组已解散")
	}
	if group.IsFull() {
		return nil, errors.NewBusiness(errors.CodeGroupMemberLimitReached, "群成员已达上限")
	}

	isMember, err := s.memberRepo.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if isMember {
		return nil, errors.NewBusiness(errors.CodeAlreadyGroupMember, "你已是该群成员")
	}

	// 复用 JoinGroup 中的验证逻辑
	joinVerify := s.getGroupJoinVerify(ctx, groupID)
	if joinVerify {
		// 检查是否已有待处理申请
		existingReq, _ := s.joinRequestRepo.GetExistingRequest(ctx, groupID, userID)
		if existingReq != nil {
			return &dto.JoinGroupByQRCodeResponse{
				Joined:     false,
				GroupID:    groupID,
				NeedVerify: true,
				RequestID:  &existingReq.ID,
			}, nil
		}

		request := &model.GroupJoinRequest{
			GroupID:   groupID,
			UserID:    userID,
			Status:    model.JoinRequestStatusPending,
			CreatedAt: time.Now(),
		}
		if err := s.joinRequestRepo.Create(ctx, request); err != nil {
			return nil, err
		}
		return &dto.JoinGroupByQRCodeResponse{
			Joined:     false,
			GroupID:    groupID,
			NeedVerify: true,
			RequestID:  &request.ID,
		}, nil
	}

	// 直接加入
	err = s.db.Transaction(func(tx *gorm.DB) error {
		memberRepoTx := s.memberRepo.WithTx(tx)
		groupRepoTx := s.groupRepo.WithTx(tx)

		if err := memberRepoTx.AddMember(ctx, &model.GroupMember{
			GroupID:  groupID,
			UserID:   userID,
			Role:     model.GroupRoleMember,
			JoinedAt: time.Now(),
		}); err != nil {
			return err
		}
		return groupRepoTx.UpdateMemberCount(ctx, groupID, 1)
	})
	if err != nil {
		logger.Error("Failed to join group by qrcode", zap.Error(err))
		return nil, err
	}

	s.publishMemberJoinedNotification(groupID, userID, qr.CreatedBy)

	return &dto.JoinGroupByQRCodeResponse{
		Joined:     true,
		GroupID:    groupID,
		NeedVerify: false,
	}, nil
}

// createNewQRCode 生成并持久化一条新二维码记录
func (s *groupServiceImpl) createNewQRCode(ctx context.Context, userID, groupID string) (*dto.GroupQRCodeResponse, error) {
	qr := &model.GroupQRCode{
		GroupID:   groupID,
		Token:     uuid.NewString(),
		CreatedBy: userID,
		ExpireAt:  time.Now().Add(model.DefaultQRCodeTTL),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.qrcodeRepo.Create(ctx, qr); err != nil {
		logger.Error("Failed to create group qrcode", zap.Error(err))
		return nil, err
	}
	return buildQRCodeResponse(qr), nil
}

func buildQRCodeResponse(qr *model.GroupQRCode) *dto.GroupQRCodeResponse {
	return &dto.GroupQRCodeResponse{
		Token:    qr.Token,
		DeepLink: fmt.Sprintf("anychat://group/join?token=%s", qr.Token),
		ExpireAt: qr.ExpireAt.Unix(),
	}
}

func (s *groupServiceImpl) getGroupJoinVerify(ctx context.Context, groupID string) bool {	settings, err := s.settingRepo.GetSettings(ctx, groupID)
	if err != nil {
		return true
	}
	return settings.JoinVerify
}

func (s *groupServiceImpl) getPinnedMessageContent(ctx context.Context, groupID, messageID string) (string, error) {
	if s.messageClient == nil {
		return "", nil
	}

	msg, err := s.messageClient.GetMessageById(ctx, &messagepb.GetMessageByIdRequest{
		MessageId: messageID,
	})
	if err != nil {
		logger.Error("Failed to load pinned message content", zap.String("messageId", messageID), zap.Error(err))
		return "", errors.NewBusiness(errors.CodeMessageNotFound, "消息不存在")
	}

	if msg.GetConversationType() != "group" || msg.GetConversationId() != groupID {
		return "", errors.NewBusiness(errors.CodeParamError, "消息不属于该群")
	}

	return msg.GetContent(), nil
}

func (s *groupServiceImpl) sendGroupSystemMessage(ctx context.Context, groupID, operatorID, text string) error {
	if s.messageClient == nil {
		return nil
	}

	content, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return err
	}

	_, err = s.messageClient.SendMessage(ctx, &messagepb.SendMessageRequest{
		SenderId:       operatorID,
		ConversationId: groupID,
		ContentType:    "text",
		Content:        string(content),
		LocalId:        uuid.NewString(),
	})
	return err
}

// publishMemberJoinedNotification 发布成员加入通知
func (s *groupServiceImpl) publishMemberJoinedNotification(groupID, userID, inviterID string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"group_id":        groupID,
		"user_id":         userID,
		"inviter_user_id": inviterID,
		"joined_at":       time.Now().Unix(),
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
	if req.Description != nil {
		updatedFields = append(updatedFields, "description")
		payload["description"] = *req.Description
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

func (s *groupServiceImpl) publishMemberMutedNotification(groupID, operatorID, targetUserID string, mutedUntil *time.Time) {
	if s.notificationPub == nil {
		return
	}

	var mutedUntilUnix int64
	if mutedUntil != nil {
		mutedUntilUnix = mutedUntil.Unix()
	}

	payload := map[string]interface{}{
		"group_id":         groupID,
		"target_user_id":   targetUserID,
		"operator_user_id": operatorID,
		"muted_until":      mutedUntilUnix,
		"updated_at":       time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeGroupMemberMuted,
		operatorID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToGroup(groupID, notif); err != nil {
		logger.Error("Failed to publish member muted notification",
			zap.String("groupId", groupID),
			zap.Error(err))
	}
}

func (s *groupServiceImpl) publishMemberUnmutedNotification(groupID, operatorID, targetUserID string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"group_id":         groupID,
		"target_user_id":   targetUserID,
		"operator_user_id": operatorID,
		"updated_at":       time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeGroupMemberUnmuted,
		operatorID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToGroup(groupID, notif); err != nil {
		logger.Error("Failed to publish member unmuted notification",
			zap.String("groupId", groupID),
			zap.Error(err))
	}
}

// publishGroupMutedNotification 发布全体禁言通知
func (s *groupServiceImpl) publishGroupMutedNotification(groupID, operatorID string, enabled bool) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"group_id":         groupID,
		"operator_user_id": operatorID,
		"enabled":          enabled,
		"updated_at":       time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeGroupMuted,
		operatorID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToGroup(groupID, notif); err != nil {
		logger.Error("Failed to publish group muted notification",
			zap.String("groupId", groupID),
			zap.Error(err))
	}
}

func (s *groupServiceImpl) publishGroupSettingsUpdatedNotification(groupID, operatorID string, updates map[string]any) {
	if s.notificationPub == nil {
		return
	}

	updatedFields := make([]string, 0, len(updates))
	for key := range updates {
		updatedFields = append(updatedFields, key)
	}

	payload := map[string]interface{}{
		"group_id":         groupID,
		"operator_user_id": operatorID,
		"updated_fields":   updatedFields,
		"updated_at":       time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeGroupSettingsUpdated,
		operatorID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToGroup(groupID, notif); err != nil {
		logger.Error("Failed to publish group settings updated notification",
			zap.String("groupId", groupID),
			zap.Error(err))
	}
}

func (s *groupServiceImpl) publishGroupMessagePinnedNotification(groupID, operatorID, messageID string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"group_id":         groupID,
		"operator_user_id": operatorID,
		"message_id":       messageID,
		"pinned_at":        time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeGroupMessagePinned,
		operatorID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToGroup(groupID, notif); err != nil {
		logger.Error("Failed to publish group message pinned notification",
			zap.String("groupId", groupID),
			zap.Error(err))
	}
}

func (s *groupServiceImpl) publishGroupMessageUnpinnedNotification(groupID, operatorID, messageID string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"group_id":         groupID,
		"operator_user_id": operatorID,
		"message_id":       messageID,
		"unpinned_at":      time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeGroupMessageUnpinned,
		operatorID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToGroup(groupID, notif); err != nil {
		logger.Error("Failed to publish group message unpinned notification",
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
