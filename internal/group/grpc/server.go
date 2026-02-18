package grpc

import (
	"context"

	commonpb "github.com/anychat/server/api/proto/common"
	grouppb "github.com/anychat/server/api/proto/group"
	"github.com/anychat/server/internal/group/dto"
	"github.com/anychat/server/internal/group/service"
	"github.com/anychat/server/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GroupServer group gRPC服务器
type GroupServer struct {
	grouppb.UnimplementedGroupServiceServer
	groupService service.GroupService
}

// NewGroupServer 创建group gRPC服务器
func NewGroupServer(groupService service.GroupService) *GroupServer {
	return &GroupServer{
		groupService: groupService,
	}
}

// ========== 内部服务调用接口实现 ==========

// GetGroupInfo 获取群信息
func (s *GroupServer) GetGroupInfo(ctx context.Context, req *grouppb.GetGroupInfoRequest) (*grouppb.GetGroupInfoResponse, error) {
	// 使用空userID调用（内部调用不需要权限检查）
	resp, err := s.groupService.GetGroupInfo(ctx, "", req.GroupId)
	if err != nil {
		return nil, convertError(err)
	}

	return &grouppb.GetGroupInfoResponse{
		GroupId:      resp.GroupID,
		Name:         resp.Name,
		Avatar:       resp.Avatar,
		Announcement: resp.Announcement,
		OwnerId:      resp.OwnerID,
		MemberCount:  resp.MemberCount,
		MaxMembers:   resp.MaxMembers,
		JoinVerify:   resp.JoinVerify,
		IsMuted:      resp.IsMuted,
		Status:       int32(1), // active
		CreatedAt:    timestamppb.New(resp.CreatedAt),
		UpdatedAt:    timestamppb.New(resp.UpdatedAt),
	}, nil
}

// GetGroupMembers 获取群成员列表
func (s *GroupServer) GetGroupMembers(ctx context.Context, req *grouppb.GetGroupMembersRequest) (*grouppb.GetGroupMembersResponse, error) {
	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	resp, err := s.groupService.GetGroupMembers(ctx, req.UserId, req.GroupId, page, pageSize)
	if err != nil {
		return nil, convertError(err)
	}

	// DTO -> Proto 转换
	members := make([]*grouppb.GroupMember, 0, len(resp.Members))
	for _, m := range resp.Members {
		member := &grouppb.GroupMember{
			UserId:   m.UserID,
			Role:     m.Role,
			IsMuted:  m.IsMuted,
			JoinedAt: timestamppb.New(m.JoinedAt),
		}
		if m.GroupNickname != nil {
			member.GroupNickname = m.GroupNickname
		}
		if m.UserInfo != nil {
			member.UserInfo = &commonpb.UserInfo{
				UserId:   m.UserInfo.UserID,
				Nickname: m.UserInfo.Nickname,
				Avatar:   m.UserInfo.Avatar,
			}
		}
		members = append(members, member)
	}

	return &grouppb.GetGroupMembersResponse{
		Members: members,
		Total:   resp.Total,
	}, nil
}

// IsMember 检查是否为群成员
func (s *GroupServer) IsMember(ctx context.Context, req *grouppb.IsMemberRequest) (*grouppb.IsMemberResponse, error) {
	isMember, role, err := s.groupService.IsMember(ctx, req.GroupId, req.UserId)
	if err != nil {
		return nil, convertError(err)
	}

	return &grouppb.IsMemberResponse{
		IsMember: isMember,
		Role:     role,
	}, nil
}

// GetUserGroups 获取用户加入的群列表
func (s *GroupServer) GetUserGroups(ctx context.Context, req *grouppb.GetUserGroupsRequest) (*grouppb.GetUserGroupsResponse, error) {
	var lastUpdateTime *int64
	if req.LastUpdateTime != nil {
		lastUpdateTime = req.LastUpdateTime
	}

	resp, err := s.groupService.GetUserGroups(ctx, req.UserId, lastUpdateTime)
	if err != nil {
		return nil, convertError(err)
	}

	// DTO -> Proto 转换
	groups := make([]*grouppb.GroupInfo, 0, len(resp.Groups))
	for _, g := range resp.Groups {
		groups = append(groups, &grouppb.GroupInfo{
			GroupId:     g.GroupID,
			Name:        g.Name,
			Avatar:      g.Avatar,
			MemberCount: g.MemberCount,
			UpdatedAt:   timestamppb.New(g.UpdatedAt),
		})
	}

	return &grouppb.GetUserGroupsResponse{
		Groups:     groups,
		UpdateTime: resp.UpdateTime,
		Total:      resp.Total,
	}, nil
}

// ========== Gateway HTTP API调用接口实现 ==========

// CreateGroup 创建群组
func (s *GroupServer) CreateGroup(ctx context.Context, req *grouppb.CreateGroupRequest) (*grouppb.CreateGroupResponse, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.CreateGroupRequest{
		Name:      req.Name,
		MemberIDs: req.MemberIds,
	}
	if req.Avatar != nil {
		dtoReq.Avatar = *req.Avatar
	}
	if req.JoinVerify != nil {
		dtoReq.JoinVerify = *req.JoinVerify
	}

	// 调用service层
	resp, err := s.groupService.CreateGroup(ctx, req.UserId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &grouppb.CreateGroupResponse{
		GroupId:     resp.GroupID,
		Name:        resp.Name,
		Avatar:      &resp.Avatar,
		OwnerId:     resp.OwnerID,
		MemberCount: resp.MemberCount,
		CreatedAt:   timestamppb.New(resp.CreatedAt),
	}, nil
}

// UpdateGroup 更新群信息
func (s *GroupServer) UpdateGroup(ctx context.Context, req *grouppb.UpdateGroupRequest) (*commonpb.Empty, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.UpdateGroupRequest{
		Name:         req.Name,
		Avatar:       req.Avatar,
		Announcement: req.Announcement,
		JoinVerify:   req.JoinVerify,
	}

	// 调用service层
	err := s.groupService.UpdateGroup(ctx, req.UserId, req.GroupId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// DissolveGroup 解散群组
func (s *GroupServer) DissolveGroup(ctx context.Context, req *grouppb.DissolveGroupRequest) (*commonpb.Empty, error) {
	err := s.groupService.DissolveGroup(ctx, req.UserId, req.GroupId)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// InviteMembers 邀请成员
func (s *GroupServer) InviteMembers(ctx context.Context, req *grouppb.InviteMembersRequest) (*commonpb.Empty, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.InviteMembersRequest{
		UserIDs: req.InviteeIds,
	}

	// 调用service层
	err := s.groupService.InviteMembers(ctx, req.UserId, req.GroupId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// RemoveMember 移除成员
func (s *GroupServer) RemoveMember(ctx context.Context, req *grouppb.RemoveMemberRequest) (*commonpb.Empty, error) {
	err := s.groupService.RemoveMember(ctx, req.UserId, req.GroupId, req.TargetUserId)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// QuitGroup 退出群组
func (s *GroupServer) QuitGroup(ctx context.Context, req *grouppb.QuitGroupRequest) (*commonpb.Empty, error) {
	err := s.groupService.QuitGroup(ctx, req.UserId, req.GroupId)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// UpdateMemberRole 更新成员角色
func (s *GroupServer) UpdateMemberRole(ctx context.Context, req *grouppb.UpdateMemberRoleRequest) (*commonpb.Empty, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.UpdateMemberRoleRequest{
		Role: req.Role,
	}

	// 调用service层
	err := s.groupService.UpdateMemberRole(ctx, req.UserId, req.GroupId, req.TargetUserId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// UpdateMemberNickname 更新群昵称
func (s *GroupServer) UpdateMemberNickname(ctx context.Context, req *grouppb.UpdateMemberNicknameRequest) (*commonpb.Empty, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.UpdateMemberNicknameRequest{
		Nickname: req.Nickname,
	}

	// 调用service层
	err := s.groupService.UpdateMemberNickname(ctx, req.UserId, req.GroupId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// TransferOwnership 转让群主
func (s *GroupServer) TransferOwnership(ctx context.Context, req *grouppb.TransferOwnershipRequest) (*commonpb.Empty, error) {
	err := s.groupService.TransferOwnership(ctx, req.UserId, req.GroupId, req.NewOwnerId)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// JoinGroup 加入群组
func (s *GroupServer) JoinGroup(ctx context.Context, req *grouppb.JoinGroupRequest) (*grouppb.JoinGroupResponse, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.JoinGroupRequest{}
	if req.Message != nil {
		dtoReq.Message = *req.Message
	}

	// 调用service层
	resp, err := s.groupService.JoinGroup(ctx, req.UserId, req.GroupId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	result := &grouppb.JoinGroupResponse{
		NeedVerify: resp.NeedVerify,
	}
	if resp.RequestID != nil {
		result.RequestId = resp.RequestID
	}

	return result, nil
}

// HandleJoinRequest 处理入群申请
func (s *GroupServer) HandleJoinRequest(ctx context.Context, req *grouppb.HandleJoinRequestRequest) (*commonpb.Empty, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.HandleJoinRequestRequest{
		Accept: req.Accept,
	}

	// 调用service层
	err := s.groupService.HandleJoinRequest(ctx, req.UserId, req.RequestId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// GetJoinRequests 获取入群申请列表
func (s *GroupServer) GetJoinRequests(ctx context.Context, req *grouppb.GetJoinRequestsRequest) (*grouppb.GetJoinRequestsResponse, error) {
	var status *string
	if req.Status != nil {
		status = req.Status
	}

	resp, err := s.groupService.GetJoinRequests(ctx, req.GroupId, status)
	if err != nil {
		return nil, convertError(err)
	}

	// DTO -> Proto 转换
	requests := make([]*grouppb.JoinRequest, 0, len(resp.Requests))
	for _, r := range resp.Requests {
		request := &grouppb.JoinRequest{
			Id:        r.ID,
			GroupId:   r.GroupID,
			UserId:    r.UserID,
			Message:   &r.Message,
			Status:    r.Status,
			CreatedAt: timestamppb.New(r.CreatedAt),
		}
		if r.InviterID != nil {
			request.InviterId = r.InviterID
		}
		if r.UserInfo != nil {
			request.UserInfo = &commonpb.UserInfo{
				UserId:   r.UserInfo.UserID,
				Nickname: r.UserInfo.Nickname,
				Avatar:   r.UserInfo.Avatar,
			}
		}
		requests = append(requests, request)
	}

	return &grouppb.GetJoinRequestsResponse{
		Requests: requests,
		Total:    resp.Total,
	}, nil
}

// UpdateGroupSettings 更新群组设置
func (s *GroupServer) UpdateGroupSettings(ctx context.Context, req *grouppb.UpdateGroupSettingsRequest) (*commonpb.Empty, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.UpdateGroupSettingsRequest{
		AllowMemberInvite:  req.AllowMemberInvite,
		AllowViewHistory:   req.AllowViewHistory,
		AllowAddFriend:     req.AllowAddFriend,
		ShowMemberNickname: req.ShowMemberNickname,
	}

	// 调用service层
	err := s.groupService.UpdateGroupSettings(ctx, req.UserId, req.GroupId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// GetGroupSettings 获取群组设置
func (s *GroupServer) GetGroupSettings(ctx context.Context, req *grouppb.GetGroupSettingsRequest) (*grouppb.GetGroupSettingsResponse, error) {
	resp, err := s.groupService.GetGroupSettings(ctx, req.GroupId)
	if err != nil {
		return nil, convertError(err)
	}

	return &grouppb.GetGroupSettingsResponse{
		GroupId:            resp.GroupID,
		AllowMemberInvite:  resp.AllowMemberInvite,
		AllowViewHistory:   resp.AllowViewHistory,
		AllowAddFriend:     resp.AllowAddFriend,
		ShowMemberNickname: resp.ShowMemberNickname,
	}, nil
}

// convertError 将业务错误转换为gRPC错误
func convertError(err error) error {
	if bizErr, ok := err.(*errors.Business); ok {
		switch bizErr.Code {
		case errors.CodeGroupNotFound:
			return status.Error(codes.NotFound, bizErr.Message)
		case errors.CodeGroupDissolved:
			return status.Error(codes.FailedPrecondition, bizErr.Message)
		case errors.CodeGroupMemberTooFew:
			return status.Error(codes.FailedPrecondition, bizErr.Message)
		case errors.CodeGroupMemberLimitReached:
			return status.Error(codes.ResourceExhausted, bizErr.Message)
		case errors.CodeNotGroupMember:
			return status.Error(codes.PermissionDenied, bizErr.Message)
		case errors.CodeAlreadyGroupMember:
			return status.Error(codes.AlreadyExists, bizErr.Message)
		case errors.CodeNoOwnerPermission:
			return status.Error(codes.PermissionDenied, bizErr.Message)
		case errors.CodeNoAdminPermission:
			return status.Error(codes.PermissionDenied, bizErr.Message)
		case errors.CodeCannotRemoveOwner:
			return status.Error(codes.PermissionDenied, bizErr.Message)
		case errors.CodeCannotRemoveAdmin:
			return status.Error(codes.PermissionDenied, bizErr.Message)
		case errors.CodeGroupNameSensitive:
			return status.Error(codes.InvalidArgument, bizErr.Message)
		case errors.CodeAnnouncementSensitive:
			return status.Error(codes.InvalidArgument, bizErr.Message)
		case errors.CodeJoinRequestNotFound:
			return status.Error(codes.NotFound, bizErr.Message)
		case errors.CodeJoinRequestProcessed:
			return status.Error(codes.FailedPrecondition, bizErr.Message)
		case errors.CodeMemberMuted:
			return status.Error(codes.PermissionDenied, bizErr.Message)
		case errors.CodeCannotQuitOwnGroup:
			return status.Error(codes.FailedPrecondition, bizErr.Message)
		case errors.CodeGroupQRExpired:
			return status.Error(codes.FailedPrecondition, bizErr.Message)
		case errors.CodeParamError:
			return status.Error(codes.InvalidArgument, bizErr.Message)
		default:
			return status.Error(codes.Internal, bizErr.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}
