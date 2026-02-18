package service

import (
	"context"
	"time"

	friendpb "github.com/anychat/server/api/proto/friend"
	grouppb "github.com/anychat/server/api/proto/group"
	messagepb "github.com/anychat/server/api/proto/message"
	sessionpb "github.com/anychat/server/api/proto/session"
	syncpb "github.com/anychat/server/api/proto/sync"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"go.uber.org/zap"
)

const defaultMsgLimit = 50

// SyncService 数据同步服务接口
type SyncService interface {
	Sync(ctx context.Context, req *syncpb.SyncRequest) (*syncpb.SyncResponse, error)
	SyncMessages(ctx context.Context, req *syncpb.SyncMessagesRequest) (*syncpb.SyncMessagesResponse, error)
}

// syncServiceImpl 同步服务实现（聚合各服务增量数据）
type syncServiceImpl struct {
	friendClient   friendpb.FriendServiceClient
	groupClient    grouppb.GroupServiceClient
	sessionClient  sessionpb.SessionServiceClient
	messageClient  messagepb.MessageServiceClient
	notificationPub notification.Publisher
}

// NewSyncService 创建同步服务
func NewSyncService(
	friendClient friendpb.FriendServiceClient,
	groupClient grouppb.GroupServiceClient,
	sessionClient sessionpb.SessionServiceClient,
	messageClient messagepb.MessageServiceClient,
	notificationPub notification.Publisher,
) SyncService {
	return &syncServiceImpl{
		friendClient:    friendClient,
		groupClient:     groupClient,
		sessionClient:   sessionClient,
		messageClient:   messageClient,
		notificationPub: notificationPub,
	}
}

// Sync 全量/增量同步：聚合 friend / group / session / message 数据
func (s *syncServiceImpl) Sync(ctx context.Context, req *syncpb.SyncRequest) (*syncpb.SyncResponse, error) {
	userID := req.UserId
	syncTime := time.Now().Unix()

	// 转换 last_sync_time：0 表示全量同步，传 nil 给下游
	var lastSyncTime *int64
	if req.LastSyncTime > 0 {
		lastSyncTime = &req.LastSyncTime
	}

	resp := &syncpb.SyncResponse{SyncTime: syncTime}

	// ── 1. 好友增量同步 ──────────────────────────────────────
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

	// ── 2. 群组增量同步 ──────────────────────────────────────
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

	// ── 3. 会话增量同步 ──────────────────────────────────────
	sessionReq := &sessionpb.GetSessionsRequest{
		UserId: userID,
		Limit:  100,
	}
	if lastSyncTime != nil {
		sessionReq.UpdatedBefore = lastSyncTime
	}
	sessionResp, err := s.sessionClient.GetSessions(ctx, sessionReq)
	if err != nil {
		logger.Warn("Sync: failed to get sessions", zap.String("userID", userID), zap.Error(err))
	} else {
		resp.Sessions = &syncpb.SyncSessionData{
			Sessions: sessionResp.Sessions,
			HasMore:  sessionResp.HasMore,
		}
	}

	// ── 4. 消息补齐（按会话序列号） ───────────────────────────
	if len(req.ConversationSeqs) > 0 {
		convMsgs, err := s.fetchConversationMessages(ctx, req.ConversationSeqs, defaultMsgLimit)
		if err != nil {
			logger.Warn("Sync: failed to fetch messages", zap.String("userID", userID), zap.Error(err))
		} else {
			resp.Conversations = convMsgs
		}
	}

	// ── 5. 发布同步完成通知（通知其他端） ────────────────────
	notif := notification.NewNotification(notification.TypeSyncCompleted, userID, notification.PriorityNormal).
		AddPayloadField("sync_time", syncTime)
	if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
		logger.Warn("Sync: failed to publish sync completed notification",
			zap.String("userID", userID), zap.Error(err))
	}

	return resp, nil
}

// SyncMessages 仅补齐各会话的离线消息
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

// fetchConversationMessages 并发拉取多个会话的新消息
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
			Reverse:        false, // 从旧到新
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
