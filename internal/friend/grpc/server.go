package grpc

import (
	"context"

	friendpb "github.com/anychat/server/api/proto/friend"
	commonpb "github.com/anychat/server/api/proto/common"
	"github.com/anychat/server/internal/friend/dto"
	"github.com/anychat/server/internal/friend/service"
	"github.com/anychat/server/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FriendServer friend gRPC服务器
type FriendServer struct {
	friendpb.UnimplementedFriendServiceServer
	friendService service.FriendService
}

// NewFriendServer 创建friend gRPC服务器
func NewFriendServer(friendService service.FriendService) *FriendServer {
	return &FriendServer{
		friendService: friendService,
	}
}

// GetFriendList 获取好友列表
func (s *FriendServer) GetFriendList(ctx context.Context, req *friendpb.GetFriendListRequest) (*friendpb.GetFriendListResponse, error) {
	// 调用service层
	resp, err := s.friendService.GetFriendList(ctx, req.UserId, req.LastUpdateTime)
	if err != nil {
		return nil, convertError(err)
	}

	// DTO -> Proto 转换
	friends := make([]*friendpb.Friend, 0, len(resp.Friends))
	for _, f := range resp.Friends {
		friend := &friendpb.Friend{
			UserId:    f.UserID,
			Remark:    f.Remark,
			CreatedAt: timestamppb.New(f.CreatedAt),
			UpdatedAt: timestamppb.New(f.UpdatedAt),
		}
		if f.UserInfo != nil {
			friend.UserInfo = &commonpb.UserInfo{
				UserId:   f.UserInfo.UserID,
				Nickname: f.UserInfo.Nickname,
				Avatar:   f.UserInfo.Avatar,
			}
		}
		friends = append(friends, friend)
	}

	return &friendpb.GetFriendListResponse{
		Friends: friends,
		Total:   resp.Total,
	}, nil
}

// SendFriendRequest 发送好友申请
func (s *FriendServer) SendFriendRequest(ctx context.Context, req *friendpb.SendFriendRequestRequest) (*friendpb.SendFriendRequestResponse, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.SendFriendRequestRequest{
		UserID:  req.ToUserId,
		Message: req.Message,
		Source:  req.Source,
	}

	// 调用service层
	resp, err := s.friendService.SendFriendRequest(ctx, req.FromUserId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &friendpb.SendFriendRequestResponse{
		RequestId:    resp.RequestID,
		AutoAccepted: resp.AutoAccepted,
	}, nil
}

// HandleFriendRequest 处理好友申请
func (s *FriendServer) HandleFriendRequest(ctx context.Context, req *friendpb.HandleFriendRequestRequest) (*commonpb.Empty, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.HandleFriendRequestRequest{
		Action: req.Action,
	}

	// 调用service层
	err := s.friendService.HandleFriendRequest(ctx, req.UserId, req.RequestId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// GetFriendRequests 获取好友申请列表
func (s *FriendServer) GetFriendRequests(ctx context.Context, req *friendpb.GetFriendRequestsRequest) (*friendpb.GetFriendRequestsResponse, error) {
	// 调用service层
	resp, err := s.friendService.GetFriendRequests(ctx, req.UserId, req.Type)
	if err != nil {
		return nil, convertError(err)
	}

	// DTO -> Proto 转换
	requests := make([]*friendpb.FriendRequest, 0, len(resp.Requests))
	for _, r := range resp.Requests {
		request := &friendpb.FriendRequest{
			Id:         r.ID,
			FromUserId: r.FromUserID,
			ToUserId:   r.ToUserID,
			Message:    r.Message,
			Source:     r.Source,
			Status:     r.Status,
			CreatedAt:  timestamppb.New(r.CreatedAt),
		}
		if r.FromUserInfo != nil {
			request.FromUserInfo = &commonpb.UserInfo{
				UserId:   r.FromUserInfo.UserID,
				Nickname: r.FromUserInfo.Nickname,
				Avatar:   r.FromUserInfo.Avatar,
			}
		}
		requests = append(requests, request)
	}

	return &friendpb.GetFriendRequestsResponse{
		Requests: requests,
		Total:    resp.Total,
	}, nil
}

// DeleteFriend 删除好友
func (s *FriendServer) DeleteFriend(ctx context.Context, req *friendpb.DeleteFriendRequest) (*commonpb.Empty, error) {
	err := s.friendService.DeleteFriend(ctx, req.UserId, req.FriendId)
	if err != nil {
		return nil, convertError(err)
	}
	return &commonpb.Empty{}, nil
}

// UpdateRemark 更新好友备注
func (s *FriendServer) UpdateRemark(ctx context.Context, req *friendpb.UpdateRemarkRequest) (*commonpb.Empty, error) {
	dtoReq := &dto.UpdateRemarkRequest{
		Remark: req.Remark,
	}

	err := s.friendService.UpdateRemark(ctx, req.UserId, req.FriendId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}
	return &commonpb.Empty{}, nil
}

// AddToBlacklist 添加到黑名单
func (s *FriendServer) AddToBlacklist(ctx context.Context, req *friendpb.AddToBlacklistRequest) (*commonpb.Empty, error) {
	dtoReq := &dto.AddToBlacklistRequest{
		UserId: req.BlockedUserId,
	}

	err := s.friendService.AddToBlacklist(ctx, req.UserId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}
	return &commonpb.Empty{}, nil
}

// RemoveFromBlacklist 从黑名单移除
func (s *FriendServer) RemoveFromBlacklist(ctx context.Context, req *friendpb.RemoveFromBlacklistRequest) (*commonpb.Empty, error) {
	err := s.friendService.RemoveFromBlacklist(ctx, req.UserId, req.BlockedUserId)
	if err != nil {
		return nil, convertError(err)
	}
	return &commonpb.Empty{}, nil
}

// GetBlacklist 获取黑名单列表
func (s *FriendServer) GetBlacklist(ctx context.Context, req *friendpb.GetBlacklistRequest) (*friendpb.GetBlacklistResponse, error) {
	resp, err := s.friendService.GetBlacklist(ctx, req.UserId)
	if err != nil {
		return nil, convertError(err)
	}

	// DTO -> Proto 转换
	items := make([]*friendpb.BlacklistItem, 0, len(resp.Items))
	for _, item := range resp.Items {
		pbItem := &friendpb.BlacklistItem{
			Id:            item.ID,
			UserId:        item.UserID,
			BlockedUserId: item.BlockedUserID,
			CreatedAt:     timestamppb.New(item.CreatedAt),
		}
		if item.BlockedUserInfo != nil {
			pbItem.BlockedUserInfo = &commonpb.UserInfo{
				UserId:   item.BlockedUserInfo.UserID,
				Nickname: item.BlockedUserInfo.Nickname,
				Avatar:   item.BlockedUserInfo.Avatar,
			}
		}
		items = append(items, pbItem)
	}

	return &friendpb.GetBlacklistResponse{
		Items: items,
		Total: resp.Total,
	}, nil
}

// IsFriend 检查是否是好友
func (s *FriendServer) IsFriend(ctx context.Context, req *friendpb.IsFriendRequest) (*friendpb.IsFriendResponse, error) {
	isFriend, err := s.friendService.IsFriend(ctx, req.UserId, req.FriendId)
	if err != nil {
		return nil, convertError(err)
	}
	return &friendpb.IsFriendResponse{
		IsFriend: isFriend,
	}, nil
}

// IsBlocked 检查是否被拉黑
func (s *FriendServer) IsBlocked(ctx context.Context, req *friendpb.IsBlockedRequest) (*friendpb.IsBlockedResponse, error) {
	isBlocked, err := s.friendService.IsBlocked(ctx, req.UserId, req.TargetUserId)
	if err != nil {
		return nil, convertError(err)
	}
	return &friendpb.IsBlockedResponse{
		IsBlocked: isBlocked,
	}, nil
}

// BatchCheckFriend 批量检查好友关系
func (s *FriendServer) BatchCheckFriend(ctx context.Context, req *friendpb.BatchCheckFriendRequest) (*friendpb.BatchCheckFriendResponse, error) {
	results, err := s.friendService.BatchCheckFriend(ctx, req.UserId, req.FriendIds)
	if err != nil {
		return nil, convertError(err)
	}
	return &friendpb.BatchCheckFriendResponse{
		Results: results,
	}, nil
}

// convertError 将业务错误转换为gRPC错误
func convertError(err error) error {
	if bizErr, ok := err.(*errors.Business); ok {
		switch bizErr.Code {
		case errors.CodeParamError:
			return status.Error(codes.InvalidArgument, bizErr.Message)
		case errors.CodeUserNotFound:
			return status.Error(codes.NotFound, bizErr.Message)
		case errors.CodeAlreadyFriend:
			return status.Error(codes.AlreadyExists, bizErr.Message)
		case errors.CodeUserBlocked:
			return status.Error(codes.PermissionDenied, bizErr.Message)
		case errors.CodeRequestExists:
			return status.Error(codes.AlreadyExists, bizErr.Message)
		case errors.CodeRequestNotFound:
			return status.Error(codes.NotFound, bizErr.Message)
		case errors.CodePermissionDenied:
			return status.Error(codes.PermissionDenied, bizErr.Message)
		case errors.CodeNotFriend:
			return status.Error(codes.FailedPrecondition, bizErr.Message)
		case errors.CodeRequestProcessed:
			return status.Error(codes.FailedPrecondition, bizErr.Message)
		default:
			return status.Error(codes.Internal, bizErr.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}
