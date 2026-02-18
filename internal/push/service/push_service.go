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

// pushTypeTitle 各推送类型的默认标题
var pushTypeTitle = map[string]string{
	notification.TypeMessageNew:      "新消息",
	notification.TypeMessageMentioned: "有人@了你",
	notification.TypeFriendRequest:   "好友申请",
	notification.TypeGroupInvited:    "群组邀请",
	notification.TypeLiveKitCallInvite: "来电",
}

// PushService 推送服务接口
type PushService interface {
	// SendPush 向指定用户列表发送推送
	SendPush(ctx context.Context, userIDs []string, title, content, pushType string, extras map[string]string) (successCount, failureCount int, msgID string, err error)
	// HandleNotification 处理来自 NATS 的通知事件
	HandleNotification(msg *nats.Msg)
}

type pushServiceImpl struct {
	jpushClient *jpush.Client
	repo        repository.PushLogRepository
}

// NewPushService 创建推送服务
func NewPushService(jpushClient *jpush.Client, repo repository.PushLogRepository) PushService {
	return &pushServiceImpl{
		jpushClient: jpushClient,
		repo:        repo,
	}
}

// SendPush 向多个用户发送推送通知
func (s *pushServiceImpl) SendPush(
	ctx context.Context,
	userIDs []string,
	title, content, pushType string,
	extras map[string]string,
) (successCount, failureCount int, msgID string, err error) {
	if len(userIDs) == 0 {
		return 0, 0, "", nil
	}

	// 批量查询 push tokens
	tokenMap, err := s.repo.GetTokensByUserIDs(userIDs)
	if err != nil {
		logger.Error("PushService: failed to get push tokens", zap.Error(err))
		return 0, len(userIDs), "", err
	}

	// 收集所有 registration_id
	var regIDs []string
	for _, rows := range tokenMap {
		for _, row := range rows {
			if row.Token != "" {
				regIDs = append(regIDs, row.Token)
			}
		}
	}

	if len(regIDs) == 0 {
		// 所有用户均无推送 token（未注册 JPush 或无设备）
		logger.Info("PushService: no push tokens found, skip push",
			zap.Strings("userIDs", userIDs))
		return 0, 0, "", nil
	}

	// 调用 JPush REST API
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

// HandleNotification 处理 NATS 通知事件，决定是否推送
func (s *pushServiceImpl) HandleNotification(msg *nats.Msg) {
	var notif notification.Notification
	if err := json.Unmarshal(msg.Data, &notif); err != nil {
		logger.Warn("PushService: failed to unmarshal notification", zap.Error(err))
		return
	}

	// 只处理需要离线推送的类型
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

// buildPushContent 根据通知类型构建推送标题；返回 (title, needPush)
func (s *pushServiceImpl) buildPushContent(notif notification.Notification) (string, bool) {
	switch notif.Type {
	case notification.TypeMessageNew,
		notification.TypeMessageMentioned,
		notification.TypeFriendRequest,
		notification.TypeGroupInvited,
		notification.TypeLiveKitCallInvite:
		title, ok := pushTypeTitle[notif.Type]
		if !ok {
			title = "新通知"
		}
		return title, true
	default:
		return "", false
	}
}

// extractContent 从通知 Payload 中提取推送正文
func (s *pushServiceImpl) extractContent(notif notification.Notification) string {
	if notif.Payload == nil {
		return ""
	}
	// 优先使用 content 字段，其次 body
	for _, key := range []string{"content", "body", "text"} {
		if v, ok := notif.Payload[key]; ok {
			if str, ok := v.(string); ok && str != "" {
				// 截断过长内容
				if len([]rune(str)) > 80 {
					str = string([]rune(str)[:80]) + "..."
				}
				return str
			}
		}
	}
	return ""
}

// extractExtras 从通知 Payload 中提取字符串附加数据
func (s *pushServiceImpl) extractExtras(notif notification.Notification) map[string]string {
	extras := map[string]string{
		"notification_type": notif.Type,
	}
	// 从 type 中解析 service（如 "message.new" → "message"）
	parts := strings.SplitN(notif.Type, ".", 2)
	if len(parts) > 0 {
		extras["service"] = parts[0]
	}
	// 复制 Payload 中的字符串字段
	for k, v := range notif.Payload {
		if str, ok := v.(string); ok {
			extras[k] = str
		}
	}
	return extras
}

// logPush 异步写入推送日志
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
