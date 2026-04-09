package service

import (
	"context"
	"fmt"
	"time"

	conversationpb "github.com/anychat/server/api/proto/conversation"
	userpb "github.com/anychat/server/api/proto/user"
	"github.com/anychat/server/internal/friend/dto"
	"github.com/anychat/server/internal/friend/model"
	"github.com/anychat/server/internal/friend/repository"
	"github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FriendService is the friend service interface
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

// friendServiceImpl is the friend service implementation
type friendServiceImpl struct {
	friendshipRepo     FriendshipRepo
	requestRepo        FriendRequestRepo
	blacklistRepo      BlacklistRepo
	userClient         userpb.UserServiceClient
	conversationClient conversationpb.ConversationServiceClient
	notificationPub    notification.Publisher
	db                 *gorm.DB
}

// FriendshipRepo is the friendship repository interface (simplified version for dependency injection)
type FriendshipRepo interface {
	repository.FriendshipRepository
}

// FriendRequestRepo is the friend request repository interface (simplified version for dependency injection)
type FriendRequestRepo interface {
	repository.FriendRequestRepository
}

// BlacklistRepo is the blacklist repository interface (simplified version for dependency injection)
type BlacklistRepo interface {
	repository.BlacklistRepository
}

// NewFriendService creates a new friend service
func NewFriendService(
	friendshipRepo repository.FriendshipRepository,
	requestRepo repository.FriendRequestRepository,
	blacklistRepo repository.BlacklistRepository,
	userClient userpb.UserServiceClient,
	conversationClient conversationpb.ConversationServiceClient,
	notificationPub notification.Publisher,
	db *gorm.DB,
) FriendService {
	return &friendServiceImpl{
		friendshipRepo:     friendshipRepo,
		requestRepo:        requestRepo,
		blacklistRepo:      blacklistRepo,
		userClient:         userClient,
		conversationClient: conversationClient,
		notificationPub:    notificationPub,
		db:                 db,
	}
}

// GetFriendList retrieves the friend list
func (s *friendServiceImpl) GetFriendList(ctx context.Context, userID string, lastUpdateTime *int64) (*dto.FriendListResponse, error) {
	var friendships []*model.Friendship
	var err error

	// Incremental sync
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

	// Convert to DTO
	friends := make([]*dto.FriendResponse, 0, len(friendships))
	for _, f := range friendships {
		friend := &dto.FriendResponse{
			UserID:    f.FriendID,
			Remark:    f.Remark,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
		}

		// Get user info (optional, failure does not affect overall result)
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

// SendFriendRequest sends a friend request
func (s *friendServiceImpl) SendFriendRequest(ctx context.Context, fromUserID string, req *dto.SendFriendRequestRequest) (*dto.SendFriendRequestResponse, error) {
	// Validation: cannot add yourself
	if fromUserID == req.UserID {
		return nil, errors.NewBusiness(errors.CodeCannotAddSelf, "")
	}

	// Check blacklist
	isBlocked, err := s.blacklistRepo.IsBlocked(ctx, fromUserID, req.UserID)
	if err != nil {
		return nil, err
	}
	if isBlocked {
		return nil, errors.NewBusiness(errors.CodeUserBlocked, "")
	}

	// Check if already a friend
	isFriend, err := s.friendshipRepo.IsFriend(ctx, fromUserID, req.UserID)
	if err != nil {
		return nil, err
	}
	if isFriend {
		return nil, errors.NewBusiness(errors.CodeAlreadyFriend, "")
	}

	// Check if there is a pending request
	existingReq, err := s.requestRepo.GetPendingRequest(ctx, fromUserID, req.UserID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if existingReq != nil {
		return nil, errors.NewBusiness(errors.CodeRequestExists, "")
	}

	// Get recipient settings, check if verification is required
	settingsResp, err := s.userClient.GetSettings(ctx, &userpb.GetSettingsRequest{UserId: req.UserID})
	if err != nil {
		logger.Error("Failed to get user settings", zap.Error(err))
		return nil, err
	}
	friendVerifyRequired := settingsResp.FriendVerifyRequired

	// Use transaction to handle request creation and possible auto-accept
	var autoAccepted bool
	err = s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// Create friend request
		status := model.FriendRequestStatusPending
		if !friendVerifyRequired {
			status = model.FriendRequestStatusAccepted
		}

		friendRequest := &model.FriendRequest{
			FromUserID: fromUserID,
			ToUserID:   req.UserID,
			Message:    req.Message,
			Source:     req.Source,
			Status:     status,
		}

		requestRepoTx := s.requestRepo.WithTx(tx)
		if err := requestRepoTx.Create(ctx, friendRequest); err != nil {
			logger.Error("Failed to create friend request", zap.Error(err))
			return err
		}

		// If recipient does not require verification, auto accept
		if !friendVerifyRequired {
			autoAccepted = true

			// Create bidirectional friendship
			friendships := []*model.Friendship{
				{
					UserID:    fromUserID,
					FriendID:  req.UserID,
					Status:    model.FriendshipStatusNormal,
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					UserID:    req.UserID,
					FriendID:  fromUserID,
					Status:    model.FriendshipStatusNormal,
					CreatedAt: now,
					UpdatedAt: now,
				},
			}

			friendshipRepoTx := s.friendshipRepo.WithTx(tx)
			if err := friendshipRepoTx.CreateBatch(ctx, friendships); err != nil {
				logger.Error("Failed to create friendship", zap.Error(err))
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Get newly created request ID
	createdReq, err := s.requestRepo.GetByUserIDs(ctx, fromUserID, req.UserID)
	if err != nil {
		logger.Error("Failed to get created friend request", zap.Error(err))
		return nil, err
	}
	requestID := createdReq.ID

	// Based on auto accept status, send different notifications
	if autoAccepted {
		// Create conversation (both sides)
		s.createFriendConversation(ctx, fromUserID, req.UserID)
		s.createFriendConversation(ctx, req.UserID, fromUserID)

		// Notify both sides
		s.publishFriendAddedNotification(fromUserID, req.UserID)
		s.publishFriendAddedNotification(req.UserID, fromUserID)
	} else {
		// Publish friend request notification
		s.publishFriendRequestNotification(createdReq)
	}

	return &dto.SendFriendRequestResponse{
		RequestID:    requestID,
		AutoAccepted: autoAccepted,
	}, nil
}

// HandleFriendRequest handles a friend request
func (s *friendServiceImpl) HandleFriendRequest(ctx context.Context, userID string, requestID int64, req *dto.HandleFriendRequestRequest) error {
	// Get request record
	friendRequest, err := s.requestRepo.GetByID(ctx, requestID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeRequestNotFound, "")
		}
		return err
	}

	// Validate permission: only recipient can handle
	if friendRequest.ToUserID != userID {
		return errors.NewBusiness(errors.CodePermissionDenied, "")
	}

	// Check request status
	if !friendRequest.IsPending() {
		return errors.NewBusiness(errors.CodeRequestProcessed, "")
	}

	// Handle request
	if req.Action == "accept" {
		// Use transaction: update request status + create bidirectional friendship
		err = s.db.Transaction(func(tx *gorm.DB) error {
			// Update request status
			requestRepoTx := s.requestRepo.WithTx(tx)
			if err := requestRepoTx.UpdateStatus(ctx, requestID, model.FriendRequestStatusAccepted); err != nil {
				return err
			}

			// Create bidirectional friendship
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

		// Publish friend request accepted notification
		s.publishFriendRequestHandledNotification(friendRequest, "accepted")
	} else if req.Action == "reject" {
		if err := s.requestRepo.UpdateStatus(ctx, requestID, model.FriendRequestStatusRejected); err != nil {
			logger.Error("Failed to reject friend request", zap.Error(err))
			return err
		}

		// Publish friend request rejected notification
		s.publishFriendRequestHandledNotification(friendRequest, "rejected")
	}

	return nil
}

// GetFriendRequests retrieves the friend request list
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

	// Convert to DTO
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

		// Get requester info
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

// DeleteFriend deletes a friend
func (s *friendServiceImpl) DeleteFriend(ctx context.Context, userID, friendID string) error {
	// Check if is a friend
	isFriend, err := s.friendshipRepo.IsFriend(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if !isFriend {
		return errors.NewBusiness(errors.CodeNotFriend, "")
	}

	// Use transaction to delete bidirectional relationship
	err = s.db.Transaction(func(tx *gorm.DB) error {
		friendshipRepoTx := s.friendshipRepo.WithTx(tx)
		return friendshipRepoTx.DeleteBidirectional(ctx, userID, friendID)
	})

	if err != nil {
		logger.Error("Failed to delete friend", zap.Error(err))
		return err
	}

	// Publish friend deleted notification
	s.publishFriendDeletedNotification(userID, friendID)

	return nil
}

// UpdateRemark updates friend remark
func (s *friendServiceImpl) UpdateRemark(ctx context.Context, userID, friendID string, req *dto.UpdateRemarkRequest) error {
	// Check if is a friend
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

	// Publish remark updated notification (multi-device sync)
	s.publishRemarkUpdatedNotification(userID, friendID, req.Remark)

	return nil
}

// AddToBlacklist adds user to blacklist
func (s *friendServiceImpl) AddToBlacklist(ctx context.Context, userID string, req *dto.AddToBlacklistRequest) error {
	// Validation: cannot block yourself
	if userID == req.UserId {
		return errors.NewBusiness(errors.CodeCannotAddSelf, "")
	}

	var removedFriend bool
	err := s.db.Transaction(func(tx *gorm.DB) error {
		blacklistRepoTx := s.blacklistRepo.WithTx(tx)
		friendshipRepoTx := s.friendshipRepo.WithTx(tx)

		// Check if already in blacklist
		existing, err := blacklistRepoTx.GetByUserAndBlocked(ctx, userID, req.UserId)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if existing != nil {
			return errors.NewBusiness(errors.CodeAlreadyInBlacklist, "")
		}

		// Create blacklist record
		blacklist := &model.Blacklist{
			UserID:        userID,
			BlockedUserID: req.UserId,
		}
		if err := blacklistRepoTx.Create(ctx, blacklist); err != nil {
			logger.Error("Failed to add to blacklist", zap.Error(err))
			return err
		}

		// If both are friends, block will automatically remove bidirectional friendship
		isFriend, err := friendshipRepoTx.IsFriend(ctx, userID, req.UserId)
		if err != nil {
			return err
		}
		if isFriend {
			if err := friendshipRepoTx.DeleteBidirectional(ctx, userID, req.UserId); err != nil {
				return err
			}
			removedFriend = true
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Publish blacklist changed notification
	s.publishBlacklistChangedNotification(userID, req.UserId, "add")

	// Trigger update on the removed friend's side
	if removedFriend {
		s.publishFriendDeletedNotification(userID, req.UserId)
	}

	return nil
}

// RemoveFromBlacklist removes user from blacklist
func (s *friendServiceImpl) RemoveFromBlacklist(ctx context.Context, userID, blockedUserID string) error {
	// Check if in blacklist
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

	// Publish blacklist changed notification
	s.publishBlacklistChangedNotification(userID, blockedUserID, "remove")

	return nil
}

// GetBlacklist retrieves the blacklist
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

		// Get blocked user info
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

// IsFriend checks if users are friends
func (s *friendServiceImpl) IsFriend(ctx context.Context, userID, friendID string) (bool, error) {
	return s.friendshipRepo.IsFriend(ctx, userID, friendID)
}

// IsBlocked checks if user is blocked
func (s *friendServiceImpl) IsBlocked(ctx context.Context, userID, targetUserID string) (bool, error) {
	return s.blacklistRepo.IsBlocked(ctx, userID, targetUserID)
}

// BatchCheckFriend batch checks friend relationships
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

// getUserInfo gets user info (internal helper method)
func (s *friendServiceImpl) getUserInfo(ctx context.Context, userID string) (*dto.UserInfo, error) {
	// Note: pass empty string as query user ID since we only need basic info
	resp, err := s.userClient.GetUserInfo(ctx, &userpb.GetUserInfoRequest{
		UserId:       "", // system internal call
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

// publishFriendRequestNotification publishes friend request notification
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

// publishFriendRequestHandledNotification publishes friend request handled notification
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

// publishFriendDeletedNotification publishes friend deleted notification
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

	// Notify the deleted friend
	if err := s.notificationPub.PublishToUser(friendID, notif); err != nil {
		logger.Error("Failed to publish friend deleted notification",
			zap.String("friendId", friendID),
			zap.Error(err))
	}
}

// publishRemarkUpdatedNotification publishes remark updated notification (multi-device sync)
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

	// Push to user's other devices (multi-device sync)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Error("Failed to publish remark updated notification",
			zap.String("userId", userID),
			zap.Error(err))
	}
}

// publishBlacklistChangedNotification publishes blacklist changed notification
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

	// Push to user's other devices (multi-device sync)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Error("Failed to publish blacklist changed notification",
			zap.String("userId", userID),
			zap.Error(err))
	}
}

// createFriendConversation creates friend conversation
func (s *friendServiceImpl) createFriendConversation(ctx context.Context, userID, friendID string) {
	if s.conversationClient == nil {
		return
	}

	_, err := s.conversationClient.CreateOrUpdateConversation(ctx, &conversationpb.CreateOrUpdateConversationRequest{
		UserId:           userID,
		ConversationType: "single",
		TargetId:         friendID,
	})
	if err != nil {
		logger.Error("Failed to create friend conversation",
			zap.String("userId", userID),
			zap.String("friendId", friendID),
			zap.Error(err))
	}
}

// publishFriendAddedNotification publishes friend added notification (auto accepted)
func (s *friendServiceImpl) publishFriendAddedNotification(userID, addedByUserID string) {
	if s.notificationPub == nil {
		return
	}

	payload := map[string]interface{}{
		"added_by_user_id": addedByUserID,
		"created_at":       time.Now().Unix(),
	}

	notif := notification.NewNotification(
		notification.TypeFriendAdded,
		userID,
		notification.PriorityNormal,
	).WithPayload(payload)

	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Error("Failed to publish friend added notification",
			zap.String("userId", userID),
			zap.String("addedByUserId", addedByUserID),
			zap.Error(err))
	}
}
