package errors

import "errors"

// 通用错误码
const (
	CodeSuccess      = 0
	CodeParamError   = 1     // 参数错误
	CodeInternalError = 2    // 内部错误
	CodeUnauthorized = 401   // 未授权
	CodeForbidden    = 403   // 禁止访问
	CodeNotFound     = 404   // 资源不存在
)

// Auth Service 错误码 (10xxx)
const (
	CodeUserExists           = 10101 // 用户已存在
	CodeVerifyCodeError      = 10102 // 验证码错误
	CodePasswordWeak         = 10103 // 密码强度不足
	CodeUserNotFound         = 10104 // 用户不存在
	CodePasswordError        = 10105 // 密码错误
	CodeAccountDisabled      = 10106 // 账号已被禁用
	CodeRefreshTokenInvalid  = 10107 // RefreshToken无效
	CodeRefreshTokenExpired  = 10108 // RefreshToken已过期
	CodeTokenInvalid         = 10109 // Token无效
	CodeTokenExpired         = 10110 // Token已过期
)

// User Service 错误码 (20xxx)
const (
	CodeNicknameUsed         = 20101 // 昵称已被使用
	CodeNicknameSensitive    = 20102 // 昵称包含敏感词
	CodeUserProfileNotFound  = 20103 // 用户不存在
	CodeQRCodeExpired        = 20104 // 二维码已过期
	CodeQRCodeInvalid        = 20105 // 二维码无效
)

// Friend Service 错误码 (30xxx)
const (
	CodeAlreadyFriend         = 30101 // 已经是好友
	CodeBlockedByUser         = 30102 // 对方已拉黑你
	CodeDuplicateRequest      = 30103 // 重复发送申请
	CodeFriendNotFound        = 30104 // 好友不存在
	CodeRequestNotFound       = 30105 // 申请不存在
	CodeCannotAddSelf         = 30106 // 不能添加自己为好友
	CodeRequestProcessed      = 30107 // 申请已处理
	CodeRequestExpired        = 30108 // 申请已过期
	CodeFriendLimitReached    = 30109 // 好友数量已达上限
	CodeTargetFriendLimit     = 30110 // 对方好友数量已达上限
	CodeBlacklistLimitReached = 30111 // 黑名单数量已达上限
	CodeAlreadyInBlacklist    = 30112 // 已在黑名单中
	CodeNotInBlacklist        = 30113 // 不在黑名单中
	CodeUserBlocked           = 30114 // 用户被拉黑
	CodeRequestExists         = 30115 // 申请已存在
	CodePermissionDenied      = 30116 // 权限不足
	CodeNotFriend             = 30117 // 不是好友
)

// Group Service 错误码 (40xxx)
const (
	CodeGroupNotFound            = 40101 // 群组不存在
	CodeGroupDissolved           = 40102 // 群组已解散
	CodeGroupMemberTooFew        = 40103 // 群成员数量不足
	CodeGroupMemberLimitReached  = 40104 // 群成员已达上限
	CodeNotGroupMember           = 40105 // 不是群成员
	CodeAlreadyGroupMember       = 40106 // 已经是群成员
	CodeNoOwnerPermission        = 40107 // 无群主权限
	CodeNoAdminPermission        = 40108 // 无管理员权限
	CodeCannotRemoveOwner        = 40109 // 不能移除群主
	CodeCannotRemoveAdmin        = 40110 // 不能移除管理员
	CodeGroupNameSensitive       = 40111 // 群名称包含敏感词
	CodeAnnouncementSensitive    = 40112 // 群公告包含敏感词
	CodeJoinRequestNotFound      = 40113 // 入群申请不存在
	CodeJoinRequestProcessed     = 40114 // 入群申请已处理
	CodeMemberMuted              = 40115 // 群内已被禁言
	CodeCannotQuitOwnGroup       = 40116 // 不能退出自己的群
	CodeGroupQRExpired           = 40117 // 群二维码已过期
)

// Message Service 错误码 (50xxx)
const (
	CodeMessageNotFound          = 50101 // 消息不存在
	CodeMessageSendFailed        = 50102 // 消息发送失败
	CodeMessageRecallFailed      = 50103 // 消息撤回失败
	CodeMessageRecallTimeLimit   = 50104 // 消息撤回时间超限
	CodeMessageDeleteFailed      = 50105 // 消息删除失败
	CodeMessagePermissionDenied  = 50106 // 消息权限不足
	CodeConversationNotFound     = 50107 // 会话不存在
	CodeSequenceGenerateFailed   = 50108 // 序列号生成失败
	CodeMarkReadFailed           = 50109 // 标记已读失败
	CodeGetUnreadCountFailed     = 50110 // 获取未读数失败
	CodeSearchMessageFailed      = 50111 // 搜索消息失败
	CodeInvalidOperation         = 50112 // 无效操作
)

// File Service 错误码 (70xxx)
const (
	CodeFileNotFound         = 70101 // 文件不存在
	CodeFileAccessDenied     = 70102 // 无权访问文件
	CodeFileSizeExceeded     = 70103 // 文件大小超限
	CodeFileTypeNotAllowed   = 70104 // 文件类型不允许
	CodeFileUploadFailed     = 70105 // 文件上传失败
	CodeFileAlreadyExists    = 70106 // 文件已存在
	CodeInvalidFileID        = 70107 // 无效的文件ID
	CodeFileExpired          = 70108 // 文件已过期
	CodeStorageQuotaExceeded = 70109 // 存储空间不足
	CodeThumbnailGenFailed   = 70110 // 缩略图生成失败
)

// Sync Service 错误码 (11xxx)
const (
	CodeSyncFailed         = 11101 // 同步失败
	CodeSyncMessagesFailed = 11102 // 消息补齐失败
)

// Push Service 错误码 (80xxx)
const (
	CodePushFailed          = 80101 // 推送失败
	CodePushTokenNotFound   = 80102 // 推送 Token 不存在
	CodePushConfigInvalid   = 80103 // 推送配置无效
)

// RTC Service 错误码 (90xxx)
const (
	CodeCallNotFound          = 90101 // 通话不存在
	CodeCallAlreadyActive     = 90102 // 通话已进行中
	CodeCallPermissionDenied  = 90103 // 无权操作此通话
	CodeCallInvalidStatus     = 90104 // 通话状态无效
	CodeMeetingNotFound       = 90105 // 会议室不存在
	CodeMeetingPasswordWrong  = 90106 // 会议室密码错误
	CodeMeetingAlreadyEnded   = 90107 // 会议室已结束
	CodeMeetingPermission     = 90108 // 无权操作此会议室
	CodeLiveKitTokenFailed    = 90109 // LiveKit Token 生成失败
	CodeLiveKitRoomFailed     = 90110 // LiveKit 房间操作失败
)

// Admin Service 错误码 (12xxx)
const (
	CodeAdminNotFound        = 12101 // 管理员不存在
	CodeAdminDisabled        = 12102 // 管理员账号已禁用
	CodeAdminUsernameExists  = 12103 // 管理员用户名已存在
	CodeAdminInvalidPassword = 12104 // 管理员密码错误
	CodeAdminTokenInvalid    = 12105 // 管理员Token无效
	CodeAdminPermission      = 12106 // 管理员权限不足
	CodeConfigKeyNotFound    = 12107 // 配置项不存在
)

// Session Service 错误码 (60xxx)
const (
	CodeSessionNotFound      = 60101 // 会话不存在
	CodeSessionDeleted       = 60102 // 会话已删除
	CodeSessionCreateFailed  = 60103 // 会话创建失败
	CodeUnreadCountFailed    = 60104 // 未读数统计错误
)

// 错误消息映射
var errorMessages = map[int]string{
	CodeSuccess:             "成功",
	CodeParamError:          "参数错误",
	CodeInternalError:       "内部错误",
	CodeUnauthorized:        "未授权",
	CodeForbidden:           "禁止访问",
	CodeNotFound:            "资源不存在",

	CodeUserExists:          "用户已存在",
	CodeVerifyCodeError:     "验证码错误",
	CodePasswordWeak:        "密码强度不足",
	CodeUserNotFound:        "用户不存在",
	CodePasswordError:       "密码错误",
	CodeAccountDisabled:     "账号已被禁用",
	CodeRefreshTokenInvalid: "RefreshToken无效",
	CodeRefreshTokenExpired: "RefreshToken已过期",
	CodeTokenInvalid:        "Token无效",
	CodeTokenExpired:        "Token已过期",

	CodeNicknameUsed:        "昵称已被使用",
	CodeNicknameSensitive:   "昵称包含敏感词",
	CodeUserProfileNotFound: "用户不存在",
	CodeQRCodeExpired:       "二维码已过期",
	CodeQRCodeInvalid:       "二维码无效",

	CodeAlreadyFriend:         "已经是好友",
	CodeBlockedByUser:         "对方已拉黑你",
	CodeDuplicateRequest:      "重复发送申请",
	CodeFriendNotFound:        "好友不存在",
	CodeRequestNotFound:       "申请不存在",
	CodeCannotAddSelf:         "不能添加自己为好友",
	CodeRequestProcessed:      "申请已处理",
	CodeRequestExpired:        "申请已过期",
	CodeFriendLimitReached:    "好友数量已达上限",
	CodeTargetFriendLimit:     "对方好友数量已达上限",
	CodeBlacklistLimitReached: "黑名单数量已达上限",
	CodeAlreadyInBlacklist:    "已在黑名单中",
	CodeNotInBlacklist:        "不在黑名单中",
	CodeUserBlocked:           "用户被拉黑",
	CodeRequestExists:         "申请已存在",
	CodePermissionDenied:      "权限不足",
	CodeNotFriend:             "不是好友",

	CodeGroupNotFound:           "群组不存在",
	CodeGroupDissolved:          "群组已解散",
	CodeGroupMemberTooFew:       "群成员数量不足",
	CodeGroupMemberLimitReached: "群成员已达上限",
	CodeNotGroupMember:          "不是群成员",
	CodeAlreadyGroupMember:      "已经是群成员",
	CodeNoOwnerPermission:       "无群主权限",
	CodeNoAdminPermission:       "无管理员权限",
	CodeCannotRemoveOwner:       "不能移除群主",
	CodeCannotRemoveAdmin:       "不能移除管理员",
	CodeGroupNameSensitive:      "群名称包含敏感词",
	CodeAnnouncementSensitive:   "群公告包含敏感词",
	CodeJoinRequestNotFound:     "入群申请不存在",
	CodeJoinRequestProcessed:    "入群申请已处理",
	CodeMemberMuted:             "群内已被禁言",
	CodeCannotQuitOwnGroup:      "不能退出自己的群",
	CodeGroupQRExpired:          "群二维码已过期",

	CodeMessageNotFound:         "消息不存在",
	CodeMessageSendFailed:       "消息发送失败",
	CodeMessageRecallFailed:     "消息撤回失败",
	CodeMessageRecallTimeLimit:  "消息撤回时间超限",
	CodeMessageDeleteFailed:     "消息删除失败",
	CodeMessagePermissionDenied: "消息权限不足",
	CodeConversationNotFound:    "会话不存在",
	CodeSequenceGenerateFailed:  "序列号生成失败",
	CodeMarkReadFailed:          "标记已读失败",
	CodeGetUnreadCountFailed:    "获取未读数失败",
	CodeSearchMessageFailed:     "搜索消息失败",
	CodeInvalidOperation:        "无效操作",

	CodeFileNotFound:         "文件不存在",
	CodeFileAccessDenied:     "无权访问文件",
	CodeFileSizeExceeded:     "文件大小超限",
	CodeFileTypeNotAllowed:   "文件类型不允许",
	CodeFileUploadFailed:     "文件上传失败",
	CodeFileAlreadyExists:    "文件已存在",
	CodeInvalidFileID:        "无效的文件ID",
	CodeFileExpired:          "文件已过期",
	CodeStorageQuotaExceeded: "存储空间不足",
	CodeThumbnailGenFailed:   "缩略图生成失败",

	CodeSessionNotFound:     "会话不存在",
	CodeSessionDeleted:      "会话已删除",
	CodeSessionCreateFailed: "会话创建失败",
	CodeUnreadCountFailed:   "未读数统计错误",

	CodeSyncFailed:         "同步失败",
	CodeSyncMessagesFailed: "消息补齐失败",

	CodePushFailed:        "推送失败",
	CodePushTokenNotFound: "推送 Token 不存在",
	CodePushConfigInvalid: "推送配置无效",

	CodeCallNotFound:         "通话不存在",
	CodeCallAlreadyActive:    "通话已进行中",
	CodeCallPermissionDenied: "无权操作此通话",
	CodeCallInvalidStatus:    "通话状态无效",
	CodeMeetingNotFound:      "会议室不存在",
	CodeMeetingPasswordWrong: "会议室密码错误",
	CodeMeetingAlreadyEnded:  "会议室已结束",
	CodeMeetingPermission:    "无权操作此会议室",
	CodeLiveKitTokenFailed:   "LiveKit Token 生成失败",
	CodeLiveKitRoomFailed:    "LiveKit 房间操作失败",

	CodeAdminNotFound:        "管理员不存在",
	CodeAdminDisabled:        "管理员账号已禁用",
	CodeAdminUsernameExists:  "管理员用户名已存在",
	CodeAdminInvalidPassword: "管理员密码错误",
	CodeAdminTokenInvalid:    "管理员Token无效",
	CodeAdminPermission:      "管理员权限不足",
	CodeConfigKeyNotFound:    "配置项不存在",
}

// GetMessage 获取错误消息
func GetMessage(code int) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	}
	return "未知错误"
}

// Business 业务错误
type Business struct {
	Code    int
	Message string
}

func (e *Business) Error() string {
	return e.Message
}

// NewBusiness 创建业务错误
func NewBusiness(code int, message string) error {
	if message == "" {
		message = GetMessage(code)
	}
	return &Business{
		Code:    code,
		Message: message,
	}
}

// IsBusiness 判断是否为业务错误
func IsBusiness(err error) bool {
	var bizErr *Business
	return errors.As(err, &bizErr)
}

// GetBusinessCode 获取业务错误码
func GetBusinessCode(err error) int {
	var bizErr *Business
	if errors.As(err, &bizErr) {
		return bizErr.Code
	}
	return CodeInternalError
}
