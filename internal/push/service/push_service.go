package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/anychat/server/internal/push/jpush"
	"github.com/anychat/server/internal/push/model"
	"github.com/anychat/server/internal/push/repository"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// pushTypeTitle default titles for each push type
var pushTypeTitle = map[string]string{
	notification.TypeMessageNew:        "New Message",
	notification.TypeMessageMentioned:  "You were mentioned",
	notification.TypeFriendRequest:     "Friend Request",
	notification.TypeGroupInvited:      "Group Invitation",
	notification.TypeLiveKitCallInvite: "Incoming Call",
}

// PushService push service interface
type PushService interface {
	// SendPush sends push to specified user list
	SendPush(ctx context.Context, userIDs []string, title, content, pushType string, extras map[string]string) (successCount, failureCount int, msgID string, err error)
	// HandleNotification handles NATS notification events
	HandleNotification(msg *nats.Msg)
}

type pushServiceImpl struct {
	jpushClient *jpush.Client
	repo        repository.PushLogRepository
}

// NewPushService creates push service
func NewPushService(jpushClient *jpush.Client, repo repository.PushLogRepository) PushService {
	return &pushServiceImpl{
		jpushClient: jpushClient,
		repo:        repo,
	}
}

// SendPush sends push notification to multiple users
func (s *pushServiceImpl) SendPush(
	ctx context.Context,
	userIDs []string,
	title, content, pushType string,
	extras map[string]string,
) (successCount, failureCount int, msgID string, err error) {
	if len(userIDs) == 0 {
		return 0, 0, "", nil
	}

	// Batch query push tokens
	tokenMap, err := s.repo.GetTokensByUserIDs(userIDs)
	if err != nil {
		logger.Error("PushService: failed to get push tokens", zap.Error(err))
		return 0, len(userIDs), "", err
	}

	// Collect all registration_id
	var regIDs []string
	for _, rows := range tokenMap {
		for _, row := range rows {
			if row.Token != "" {
				regIDs = append(regIDs, row.Token)
			}
		}
	}

	if len(regIDs) == 0 {
		// All users have no push tokens (not registered JPush or no device)
		logger.Info("PushService: no push tokens found, skip push",
			zap.Strings("userIDs", userIDs))
		return 0, 0, "", nil
	}

	// Call JPush REST API
	result, pushErr := s.jpushClient.PushToRegistrationIDs(regIDs, title, content, extras)
	if pushErr != nil {
		logger.Error("PushService: JPush request failed", zap.Error(pushErr))
		failureCount = len(regIDs)
		s.logPush(userIDs[0], pushType, title, content, len(regIDs), 0, failureCount, "", "failed", pushErr.Error())
		return 0, failureCount, "", pushErr
	}

	successCount = len(regIDs)
	s.logPush(userIDs[0], pushType, title, content, len(regIDs), successCount, 0, result.MsgID, "sent", "")

	logger.Info("PushService: push sent",
		zap.Strings("userIDs", userIDs),
		zap.String("pushType", pushType),
		zap.Int("regIDCount", len(regIDs)),
		zap.String("msgID", result.MsgID))

	return successCount, 0, result.MsgID, nil
}

// HandleNotification handles NATS notification events, decides whether to push
func (s *pushServiceImpl) HandleNotification(msg *nats.Msg) {
	var notif notification.Notification
	if err := json.Unmarshal(msg.Data, &notif); err != nil {
		logger.Warn("PushService: failed to unmarshal notification", zap.Error(err))
		return
	}

	// Only handle types that need offline push
	title, needPush := s.buildPushContent(notif)
	if !needPush {
		return
	}

	if notif.ToUserID == "" {
		return
	}

	content := s.extractContent(notif)
	extras := s.extractExtras(notif)

	s.SendPush(context.Background(), //nolint:errcheck
		[]string{notif.ToUserID},
		title, content, notif.Type,
		extras,
	)
}

// buildPushContent builds push title based on notification type; returns (title, needPush)
func (s *pushServiceImpl) buildPushContent(notif notification.Notification) (string, bool) {
	switch notif.Type {
	case notification.TypeMessageNew,
		notification.TypeMessageMentioned,
		notification.TypeFriendRequest,
		notification.TypeGroupInvited,
		notification.TypeLiveKitCallInvite:
		title, ok := pushTypeTitle[notif.Type]
		if !ok {
			title = "New Notification"
		}
		return title, true
	default:
		return "", false
	}
}

// extractContent extracts push body from notification Payload
func (s *pushServiceImpl) extractContent(notif notification.Notification) string {
	if notif.Payload == nil {
		return ""
	}
	// Prefer content field, then body
	for _, key := range []string{"content", "body", "text"} {
		if v, ok := notif.Payload[key]; ok {
			if str, ok := v.(string); ok && str != "" {
				// Truncate too long content
				if len([]rune(str)) > 80 {
					str = string([]rune(str)[:80]) + "..."
				}
				return str
			}
		}
	}
	return ""
}

// extractExtras extracts string extra data from notification Payload
func (s *pushServiceImpl) extractExtras(notif notification.Notification) map[string]string {
	extras := map[string]string{
		"notification_type": notif.Type,
	}
	// Parse service from type (e.g., "message.new" → "message")
	parts := strings.SplitN(notif.Type, ".", 2)
	if len(parts) > 0 {
		extras["service"] = parts[0]
	}
	// Copy string fields from Payload
	for k, v := range notif.Payload {
		if str, ok := v.(string); ok {
			extras[k] = str
		}
	}
	return extras
}

// logPush asynchronously writes push log
func (s *pushServiceImpl) logPush(userID, pushType, title, content string,
	targetCount, successCount, failureCount int,
	jpushMsgID, status, errMsg string,
) {
	log := &model.PushLog{
		UserID:       userID,
		PushType:     pushType,
		Title:        title,
		Content:      content,
		TargetCount:  targetCount,
		SuccessCount: successCount,
		FailureCount: failureCount,
		JPushMsgID:   jpushMsgID,
		Status:       status,
		ErrorMsg:     errMsg,
	}
	if err := s.repo.Create(log); err != nil {
		logger.Warn("PushService: failed to write push log", zap.Error(err))
	}
}
