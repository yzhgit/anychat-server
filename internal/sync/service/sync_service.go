package service

import (
	"context"
	"time"

	conversationpb "github.com/anychat/server/api/proto/conversation"
	friendpb "github.com/anychat/server/api/proto/friend"
	grouppb "github.com/anychat/server/api/proto/group"
	messagepb "github.com/anychat/server/api/proto/message"
	syncpb "github.com/anychat/server/api/proto/sync"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"go.uber.org/zap"
)

const defaultMsgLimit = 50

// SyncService data sync service interface
type SyncService interface {
	Sync(ctx context.Context, req *syncpb.SyncRequest) (*syncpb.SyncResponse, error)
	SyncMessages(ctx context.Context, req *syncpb.SyncMessagesRequest) (*syncpb.SyncMessagesResponse, error)
}

// syncServiceImpl sync service implementation (aggregates incremental data from each service)
type syncServiceImpl struct {
	friendClient       friendpb.FriendServiceClient
	groupClient        grouppb.GroupServiceClient
	conversationClient conversationpb.ConversationServiceClient
	messageClient      messagepb.MessageServiceClient
	notificationPub    notification.Publisher
}

// NewSyncService creates sync service
func NewSyncService(
	friendClient friendpb.FriendServiceClient,
	groupClient grouppb.GroupServiceClient,
	conversationClient conversationpb.ConversationServiceClient,
	messageClient messagepb.MessageServiceClient,
	notificationPub notification.Publisher,
) SyncService {
	return &syncServiceImpl{
		friendClient:       friendClient,
		groupClient:        groupClient,
		conversationClient: conversationClient,
		messageClient:      messageClient,
		notificationPub:    notificationPub,
	}
}

// Sync full/incremental sync: aggregates friend/group/conversation/message data
func (s *syncServiceImpl) Sync(ctx context.Context, req *syncpb.SyncRequest) (*syncpb.SyncResponse, error) {
	userID := req.UserId
	syncTime := time.Now().Unix()

	// Convert last_sync_time: 0 means full sync, pass nil to downstream
	var lastSyncTime *int64
	if req.LastSyncTime > 0 {
		lastSyncTime = &req.LastSyncTime
	}

	resp := &syncpb.SyncResponse{SyncTime: syncTime}

	// ── 1. Friend incremental sync ──────────────────────────────────────
	friendResp, err := s.friendClient.GetFriendList(ctx, &friendpb.GetFriendListRequest{
		UserId:         userID,
		LastUpdateTime: lastSyncTime,
	})
	if err != nil {
		logger.Warn("Sync: failed to get friend list", zap.String("userID", userID), zap.Error(err))
	} else {
		resp.Friends = &syncpb.SyncFriendData{
			Friends: friendResp.Friends,
			Total:   friendResp.Total,
		}
	}

	// ── 2. Group incremental sync ──────────────────────────────────────
	groupResp, err := s.groupClient.GetUserGroups(ctx, &grouppb.GetUserGroupsRequest{
		UserId:         userID,
		LastUpdateTime: lastSyncTime,
	})
	if err != nil {
		logger.Warn("Sync: failed to get user groups", zap.String("userID", userID), zap.Error(err))
	} else {
		resp.Groups = &syncpb.SyncGroupData{
			Groups: groupResp.Groups,
			Total:  groupResp.Total,
		}
	}

	// ── 3. Conversation incremental sync ──────────────────────────────────────
	conversationReq := &conversationpb.GetConversationsRequest{
		UserId: userID,
		Limit:  100,
	}
	if lastSyncTime != nil {
		conversationReq.UpdatedBefore = lastSyncTime
	}
	conversationResp, err := s.conversationClient.GetConversations(ctx, conversationReq)
	if err != nil {
		logger.Warn("Sync: failed to get conversations", zap.String("userID", userID), zap.Error(err))
	} else {
		resp.ConversationData = &syncpb.SyncConversationData{
			Conversations: conversationResp.Conversations,
			HasMore:       conversationResp.HasMore,
		}
	}

	// ── 4. Message backfill (by conversation sequence) ───────────────────────────
	if len(req.ConversationSeqs) > 0 {
		convMsgs, err := s.fetchConversationMessages(ctx, req.ConversationSeqs, defaultMsgLimit)
		if err != nil {
			logger.Warn("Sync: failed to fetch messages", zap.String("userID", userID), zap.Error(err))
		} else {
			resp.Conversations = convMsgs
		}
	}

	// ── 5. Publish sync completed notification (notify other devices) ────────────────────
	notif := notification.NewNotification(notification.TypeSyncCompleted, userID, notification.PriorityNormal).
		AddPayloadField("sync_time", syncTime)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Sync: failed to publish sync completed notification",
			zap.String("userID", userID), zap.Error(err))
	}

	return resp, nil
}

// SyncMessages only backfills offline messages for each conversation
func (s *syncServiceImpl) SyncMessages(ctx context.Context, req *syncpb.SyncMessagesRequest) (*syncpb.SyncMessagesResponse, error) {
	limit := int(req.LimitPerConversation)
	if limit <= 0 {
		limit = defaultMsgLimit
	}

	convMsgs, err := s.fetchConversationMessages(ctx, req.ConversationSeqs, limit)
	if err != nil {
		return nil, err
	}

	return &syncpb.SyncMessagesResponse{Conversations: convMsgs}, nil
}

// fetchConversationMessages concurrently fetches new messages for multiple conversations
func (s *syncServiceImpl) fetchConversationMessages(
	ctx context.Context,
	seqs []*syncpb.ConversationSeq,
	limit int,
) ([]*syncpb.ConversationMessages, error) {
	result := make([]*syncpb.ConversationMessages, 0, len(seqs))

	for _, seq := range seqs {
		startSeq := seq.LastSeq + 1
		msgResp, err := s.messageClient.GetMessages(ctx, &messagepb.GetMessagesRequest{
			ConversationId: seq.ConversationId,
			StartSeq:       &startSeq,
			Limit:          int32(limit),
			Reverse:        false, // from old to new
		})
		if err != nil {
			logger.Warn("SyncMessages: failed to get messages",
				zap.String("conversationId", seq.ConversationId),
				zap.Error(err))
			continue
		}

		result = append(result, &syncpb.ConversationMessages{
			ConversationId:   seq.ConversationId,
			ConversationType: seq.ConversationType,
			Messages:         msgResp.Messages,
			HasMore:          msgResp.HasMore,
		})
	}

	return result, nil
}
