package service

import (
	"context"
	"fmt"
	"time"

	"github.com/anychat/server/internal/friend/dto"
	"github.com/anychat/server/internal/friend/model"
	"github.com/anychat/server/internal/friend/repository"
	"github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	userpb "github.com/anychat/server/api/proto/user"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FriendService 好友服务接口
type FriendService interface {
	GetFriendList(ctx context.Context, userID string, lastUpdateTime *int64) (*dto.FriendListResponse, error)
	SendFriendRequest(ctx context.Context, fromUserID string, req *dto.SendFriendRequestRequest) (*dto.SendFriendRequestResponse, error)
	HandleFriendRequest(ctx context.Context, userID string, requestID int64, req *dto.HandleFriendRequestRequest) error
	GetFriendRequests(ctx context.Context, userID string, requestType string) (*dto.FriendRequestListResponse, error)
	DeleteFriend(ctx context.Context, userID, friendID string) error
	UpdateRemark(ctx context.Context, userID, friendID string, req *dto.UpdateRemarkRequest) error
	AddToBlacklist(ctx context.Context, userID string, req *dto.AddToBlacklistRequest) error
	RemoveFromBlacklist(ctx context.Context, userID, blockedUserID string) error
	GetBlacklist(ctx context.Context, userID string) (*dto.BlacklistResponse, error)
	IsFriend(ctx context.Context, userID, friendID string) (bool, error)
	IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error)
	BatchCheckFriend(ctx context.Context, userID string, friendIDs []string) (map[string]bool, error)
}

// friendServiceImpl 好友服务实现
type friendServiceImpl struct {
	friendshipRepo  FriendshipRepo
	requestRepo     FriendRequestRepo
	blacklistRepo   BlacklistRepo
	userClient      userpb.UserServiceClient
	notificationPub notification.Publisher
	db              *gorm.DB
}

// FriendshipRepo 好友关系仓库接口（简化版，用于依赖注入）
type FriendshipRepo interface {
	repository.FriendshipRepository
}

// FriendRequestRepo 好友申请仓库接口（简化版，用于依赖注入）
type FriendRequestRepo interface {
	repository.FriendRequestRepository
}

// BlacklistRepo 黑名单仓库接口（简化版，用于依赖注入）
type BlacklistRepo interface {
	repository.BlacklistRepository
}

// NewFriendService 创建好友服务
func NewFriendService(
	friendshipRepo repository.FriendshipRepository,
	requestRepo repository.FriendRequestRepository,
	blacklistRepo repository.BlacklistRepository,
	userClient userpb.UserServiceClient,
	notificationPub notification.Publisher,
	db *gorm.DB,
) FriendService {
	return &friendServiceImpl{
		friendshipRepo:  friendshipRepo,
		requestRepo:     requestRepo,
		blacklistRepo:   blacklistRepo,
		userClient:      userClient,
		notificationPub: notificationPub,
		db:              db,
	}
}

// GetFriendList 获取好友列表
func (s *friendServiceImpl) GetFriendList(ctx context.Context, userID string, lastUpdateTime *int64) (*dto.FriendListResponse, error) {
	var friendships []*model.Friendship
	var err error

	// 增量同步
	if lastUpdateTime != nil && *lastUpdateTime > 0 {
		t := time.Unix(*lastUpdateTime, 0)
		friendships, err = s.friendshipRepo.GetFriendListByUpdateTime(ctx, userID, t)
	} else {
		friendships, err = s.friendshipRepo.GetFriendList(ctx, userID)
	}

	if err != nil {
		logger.Error("Failed to get friend list", zap.Error(err))
		return nil, err
	}

	// 转换为DTO
	friends := make([]*dto.FriendResponse, 0, len(friendships))
	for _, f := range friendships {
		friend := &dto.FriendResponse{
			UserID:    f.FriendID,
			Remark:    f.Remark,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
		}

		// 获取用户信息（可选，如果失败不影响整体结果）
		if userInfo, err := s.getUserInfo(ctx, f.FriendID); err == nil {
			friend.UserInfo = userInfo
		}

		friends = append(friends, friend)
	}

	return &dto.FriendListResponse{
		Friends: friends,
		Total:   int64(len(friends)),
	}, nil
}

// SendFriendRequest 发送好友申请
func (s *friendServiceImpl) SendFriendRequest(ctx context.Context, fromUserID string, req *dto.SendFriendRequestRequest) (*dto.SendFriendRequestResponse, error) {
	// 验证：不能添加自己
	if fromUserID == req.UserID {
		return nil, errors.NewBusiness(errors.CodeCannotAddSelf, "")
	}

	// 检查黑名单
	isBlocked, err := s.blacklistRepo.IsBlocked(ctx, fromUserID, req.UserID)
	if err != nil {
		return nil, err
	}
	if isBlocked {
		return nil, errors.NewBusiness(errors.CodeUserBlocked, "")
	}

	// 检查是否已是好友
	isFriend, err := s.friendshipRepo.IsFriend(ctx, fromUserID, req.UserID)
	if err != nil {
		return nil, err
	}
	if isFriend {
		return nil, errors.NewBusiness(errors.CodeAlreadyFriend, "")
	}

	// 检查是否有待处理的申请
	existingReq, err := s.requestRepo.GetPendingRequest(ctx, fromUserID, req.UserID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if existingReq != nil {
		return nil, errors.NewBusiness(errors.CodeRequestExists, "")
	}

	// 创建好友申请
	friendRequest := &model.FriendRequest{
		FromUserID: fromUserID,
		ToUserID:   req.UserID,
		Message:    req.Message,
		Source:     req.Source,
		Status:     model.FriendRequestStatusPending,
	}

	if err := s.requestRepo.Create(ctx, friendRequest); err != nil {
		logger.Error("Failed to create friend request", zap.Error(err))
		return nil, err
	}

	// 发布好友请求通知
	s.publishFriendRequestNotification(friendRequest)

	// TODO: 检查对方设置是否需要验证，如果不需要则自动接受
	// 这里暂时所有申请都需要手动接受

	return &dto.SendFriendRequestResponse{
		RequestID:    friendRequest.ID,
		AutoAccepted: false,
	}, nil
}

// HandleFriendRequest 处理好友申请
func (s *friendServiceImpl) HandleFriendRequest(ctx context.Context, userID string, requestID int64, req *dto.HandleFriendRequestRequest) error {
	// 获取申请记录
	friendRequest, err := s.requestRepo.GetByID(ctx, requestID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeRequestNotFound, "")
		}
		return err
	}

	// 验证权限：只有接收方可以处理
	if friendRequest.ToUserID != userID {
		return errors.NewBusiness(errors.CodePermissionDenied, "")
	}

	// 检查申请状态
	if !friendRequest.IsPending() {
		return errors.NewBusiness(errors.CodeRequestProcessed, "")
	}

	// 处理申请
	if req.Action == "accept" {
		// 使用事务：更新申请状态 + 创建双向好友关系
		err = s.db.Transaction(func(tx *gorm.DB) error {
			// 更新申请状态
			requestRepoTx := s.requestRepo.WithTx(tx)
			if err := requestRepoTx.UpdateStatus(ctx, requestID, model.FriendRequestStatusAccepted); err != nil {
				return err
			}

			// 创建双向好友关系
			now := time.Now()
			friendships := []*model.Friendship{
				{
					UserID:    userID,
					FriendID:  friendRequest.FromUserID,
					Status:    model.FriendshipStatusNormal,
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					UserID:    friendRequest.FromUserID,
					FriendID:  userID,
					Status:    model.FriendshipStatusNormal,
					CreatedAt: now,
					UpdatedAt: now,
				},
			}

			friendshipRepoTx := s.friendshipRepo.WithTx(tx)
			return friendshipRepoTx.CreateBatch(ctx, friendships)
		})

		if err != nil {
			logger.Error("Failed to accept friend request", zap.Error(err))
			return err
		}

		// 发布好友请求接受通知
		s.publishFriendRequestHandledNotification(friendRequest, "accepted")
	} else if req.Action == "reject" {
		if err := s.requestRepo.UpdateStatus(ctx, requestID, model.FriendRequestStatusRejected); err != nil {
			logger.Error("Failed to reject friend request", zap.Error(err))
			return err
		}

		// 发布好友请求拒绝通知
		s.publishFriendRequestHandledNotification(friendRequest, "rejected")
	}

	return nil
}

// GetFriendRequests 获取好友申请列表
func (s *friendServiceImpl) GetFriendRequests(ctx context.Context, userID string, requestType string) (*dto.FriendRequestListResponse, error) {
	var requests []*model.FriendRequest
	var err error

	if requestType == "sent" {
		requests, err = s.requestRepo.GetSentRequests(ctx, userID)
	} else {
		requests, err = s.requestRepo.GetReceivedRequests(ctx, userID)
	}

	if err != nil {
		logger.Error("Failed to get friend requests", zap.Error(err))
		return nil, err
	}

	// 转换为DTO
	dtoRequests := make([]*dto.FriendRequestResponse, 0, len(requests))
	for _, r := range requests {
		dtoReq := &dto.FriendRequestResponse{
			ID:         r.ID,
			FromUserID: r.FromUserID,
			ToUserID:   r.ToUserID,
			Message:    r.Message,
			Source:     r.Source,
			Status:     r.Status,
			CreatedAt:  r.CreatedAt,
		}

		// 获取申请人信息
		if userInfo, err := s.getUserInfo(ctx, r.FromUserID); err == nil {
			dtoReq.FromUserInfo = userInfo
		}

		dtoRequests = append(dtoRequests, dtoReq)
	}

	return &dto.FriendRequestListResponse{
		Requests: dtoRequests,
		Total:    int64(len(dtoRequests)),
	}, nil
}

// DeleteFriend 删除好友
func (s *friendServiceImpl) DeleteFriend(ctx context.Context, userID, friendID string) error {
	// 检查是否是好友
	isFriend, err := s.friendshipRepo.IsFriend(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if !isFriend {
		return errors.NewBusiness(errors.CodeNotFriend, "")
	}

	// 使用事务删除双向关系
	err = s.db.Transaction(func(tx *gorm.DB) error {
		friendshipRepoTx := s.friendshipRepo.WithTx(tx)
		return friendshipRepoTx.DeleteBidirectional(ctx, userID, friendID)
	})

	if err != nil {
		logger.Error("Failed to delete friend", zap.Error(err))
		return err
	}

	// 发布好友删除通知
	s.publishFriendDeletedNotification(userID, friendID)

	return nil
}

// UpdateRemark 更新好友备注
func (s *friendServiceImpl) UpdateRemark(ctx context.Context, userID, friendID string, req *dto.UpdateRemarkRequest) error {
	// 检查是否是好友
	isFriend, err := s.friendshipRepo.IsFriend(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if !isFriend {
		return errors.NewBusiness(errors.CodeNotFriend, "")
	}

	if err := s.friendshipRepo.UpdateRemark(ctx, userID, friendID, req.Remark); err != nil {
		logger.Error("Failed to update remark", zap.Error(err))
		return err
	}

	// 发布备注更新通知（多端同步）
	s.publishRemarkUpdatedNotification(userID, friendID, req.Remark)

	return nil
}

// AddToBlacklist 添加到黑名单
func (s *friendServiceImpl) AddToBlacklist(ctx context.Context, userID string, req *dto.AddToBlacklistRequest) error {
	// 验证：不能拉黑自己
	if userID == req.UserId {
		return errors.NewBusiness(errors.CodeCannotAddSelf, "")
	}

	// 检查是否已在黑名单
	existing, err := s.blacklistRepo.GetByUserAndBlocked(ctx, userID, req.UserId)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if existing != nil {
		return errors.NewBusiness(errors.CodeAlreadyInBlacklist, "")
	}

	// 创建黑名单记录
	blacklist := &model.Blacklist{
		UserID:        userID,
		BlockedUserID: req.UserId,
	}

	if err := s.blacklistRepo.Create(ctx, blacklist); err != nil {
		logger.Error("Failed to add to blacklist", zap.Error(err))
		return err
	}

	// 发布黑名单变更通知
	s.publishBlacklistChangedNotification(userID, req.UserId, "add")

	// TODO: 如果是好友，同时删除好友关系

	return nil
}

// RemoveFromBlacklist 从黑名单移除
func (s *friendServiceImpl) RemoveFromBlacklist(ctx context.Context, userID, blockedUserID string) error {
	// 检查是否在黑名单
	existing, err := s.blacklistRepo.GetByUserAndBlocked(ctx, userID, blockedUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeNotInBlacklist, "")
		}
		return err
	}
	if existing == nil {
		return errors.NewBusiness(errors.CodeNotInBlacklist, "")
	}

	if err := s.blacklistRepo.Delete(ctx, userID, blockedUserID); err != nil {
		logger.Error("Failed to remove from blacklist", zap.Error(err))
		return err
	}

	// 发布黑名单变更通知
	s.publishBlacklistChangedNotification(userID, blockedUserID, "remove")

	return nil
}

// GetBlacklist 获取黑名单列表
func (s *friendServiceImpl) GetBlacklist(ctx context.Context, userID string) (*dto.BlacklistResponse, error) {
	blacklist, err := s.blacklistRepo.GetBlacklist(ctx, userID)
	if err != nil {
		logger.Error("Failed to get blacklist", zap.Error(err))
		return nil, err
	}

	items := make([]*dto.BlacklistItemResponse, 0, len(blacklist))
	for _, b := range blacklist {
		item := &dto.BlacklistItemResponse{
			ID:            b.ID,
			UserID:        b.UserID,
			BlockedUserID: b.BlockedUserID,
			CreatedAt:     b.CreatedAt,
		}

		// 获取被拉黑用户信息
		if userInfo, err := s.getUserInfo(ctx, b.BlockedUserID); err == nil {
			item.BlockedUserInfo = userInfo
		}

		items = append(items, item)
	}

	return &dto.BlacklistResponse{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

// IsFriend 检查是否是好友
func (s *friendServiceImpl) IsFriend(ctx context.Context, userID, friendID string) (bool, error) {
	return s.friendshipRepo.IsFriend(ctx, userID, friendID)
}

// IsBlocked 检查是否被拉黑
func (s *friendServiceImpl) IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error) {
	return s.blacklistRepo.IsBlocked(ctx, userID, targetUserID)
}

// BatchCheckFriend 批量检查好友关系
func (s *friendServiceImpl) BatchCheckFriend(ctx context.Context, userID string, friendIDs []string) (map[string]bool, error) {
	results := make(map[string]bool, len(friendIDs))

	for _, friendID := range friendIDs {
		isFriend, err := s.friendshipRepo.IsFriend(ctx, userID, friendID)
		if err != nil {
			logger.Error("Failed to check friend", zap.Error(err), zap.String("friendID", friendID))
			results[friendID] = false
			continue
		}
		results[friendID] = isFriend
	}

	return results, nil
}

// getUserInfo 获取用户信息（内部辅助方法）
func (s *friendServiceImpl) getUserInfo(ctx context.Context, userID string) (*dto.UserInfo, error) {
	// 注意：这里传空字符串作为查询者ID，因为我们只是获取基本信息
	resp, err := s.userClient.GetUserInfo(ctx, &userpb.GetUserInfoRequest{
		UserId:       "", // 系统内部调用
		TargetUserId: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	userInfo := &dto.UserInfo{
		UserID:   resp.UserId,
		Nickname: resp.Nickname,
		Avatar:   resp.Avatar,
	}

	if resp.Gender != 0 {
		gender := resp.Gender
		userInfo.Gender = &gender
	}
	if resp.Signature != "" {
		userInfo.Bio = &resp.Signature
	}

	return userInfo, nil
}

// publishFriendRequestNotification 发布好友请求通知
func (s *friendServiceImpl) publishFriendRequestNotification(req *model.FriendRequest) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"request_id":   req.ID,
		"from_user_id": req.FromUserID,
		"message":      req.Message,
		"source":       req.Source,
		"created_at":   req.CreatedAt.Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeFriendRequest,
		req.FromUserID,
		notification.PriorityHigh,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToUser(req.ToUserID, notif); err != nil {
		logger.Error("Failed to publish friend request notification",
			zap.String("toUserId", req.ToUserID),
			zap.Error(err))
	}
}

// publishFriendRequestHandledNotification 发布好友请求处理结果通知
func (s *friendServiceImpl) publishFriendRequestHandledNotification(req *model.FriendRequest, status string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"request_id": req.ID,
		"to_user_id": req.ToUserID,
		"status":     status,
		"handled_at": time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeFriendRequestHandled,
		req.ToUserID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToUser(req.FromUserID, notif); err != nil {
		logger.Error("Failed to publish friend request handled notification",
			zap.String("fromUserId", req.FromUserID),
			zap.Error(err))
	}
}

// publishFriendDeletedNotification 发布好友删除通知
func (s *friendServiceImpl) publishFriendDeletedNotification(userID, friendID string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"friend_user_id": userID,
		"deleted_at":     time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeFriendDeleted,
		userID,
		notification.PriorityNormal,
	).WithPayload(payload)

	// 通知被删除的好友
	if err := s.notificationPub.PublishToUser(friendID, notif); err != nil {
		logger.Error("Failed to publish friend deleted notification",
			zap.String("friendId", friendID),
			zap.Error(err))
	}
}

// publishRemarkUpdatedNotification 发布备注更新通知（多端同步）
func (s *friendServiceImpl) publishRemarkUpdatedNotification(userID, friendID, remark string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"friend_user_id": friendID,
		"remark":         remark,
		"updated_at":     time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeFriendRemarkUpdated,
		userID,
		notification.PriorityLow,
	).WithPayload(payload)

	// 推送给用户自己的其他设备（多端同步）
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Error("Failed to publish remark updated notification",
			zap.String("userId", userID),
			zap.Error(err))
	}
}

// publishBlacklistChangedNotification 发布黑名单变更通知
func (s *friendServiceImpl) publishBlacklistChangedNotification(userID, targetUserID, action string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"target_user_id": targetUserID,
		"action":         action,
		"changed_at":     time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeBlacklistChanged,
		userID,
		notification.PriorityNormal,
	).WithPayload(payload)

	// 推送给用户自己的其他设备（多端同步）
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Error("Failed to publish blacklist changed notification",
			zap.String("userId", userID),
			zap.Error(err))
	}
}
