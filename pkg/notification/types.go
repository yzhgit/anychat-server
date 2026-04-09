package notification

// 定义各服务的通知类型常量

// Friend Service 通知类型
const (
	TypeFriendRequest        = "friend.request"           // 好友请求
	TypeFriendRequestHandled = "friend.request_handled"   // 好友请求处理结果
	TypeFriendAdded          = "friend.added"             // 自动添加为好友（无需验证）
	TypeFriendDeleted        = "friend.deleted"           // 好友删除
	TypeFriendRemarkUpdated  = "friend.remark_updated"    // 好友备注更新
	TypeBlacklistChanged     = "friend.blacklist_changed" // 黑名单变更
)

// Group Service 通知类型
const (
	TypeGroupInvited         = "group.invited"          // 群组邀请
	TypeGroupMemberJoined    = "group.member_joined"    // 成员加入
	TypeGroupMemberLeft      = "group.member_left"      // 成员退出
	TypeGroupMemberMuted     = "group.member_muted"     // 成员禁言
	TypeGroupMemberUnmuted   = "group.member_unmuted"   // 成员解除禁言
	TypeGroupInfoUpdated     = "group.info_updated"     // 群组信息更新
	TypeGroupSettingsUpdated = "group.settings_updated" // 群设置更新
	TypeGroupRoleChanged     = "group.role_changed"     // 角色变更
	TypeGroupMuted           = "group.muted"            // 群组禁言
	TypeGroupMessagePinned   = "group.message_pinned"   // 群消息置顶
	TypeGroupMessageUnpinned = "group.message_unpinned" // 群消息取消置顶
	TypeGroupJoinRequested   = "group.join_requested"   // 入群申请
	TypeGroupQRCodeRefreshed = "group.qrcode_refreshed" // 群二维码刷新
	TypeGroupDisbanded       = "group.disbanded"        // 群组解散
)

// Message Service 通知类型
const (
	TypeMessageNew         = "message.new"          // 新消息
	TypeMessageReadReceipt = "message.read_receipt" // 已读回执
	TypeMessageRecalled    = "message.recalled"     // 消息撤回
	TypeMessageTyping      = "message.typing"       // 正在输入
	TypeMessageMentioned   = "message.mentioned"    // @提及
	TypeMessageAutoDeleted = "message.auto_deleted" // 消息自动删除
)

// User Service 通知类型
const (
	TypeUserProfileUpdated       = "user.profile_updated"        // 用户资料更新
	TypeUserFriendProfileChanged = "user.friend_profile_changed" // 好友资料变更
	TypeUserStatusChanged        = "user.status_changed"         // 在线状态变更
)

// Auth Service 通知类型
const (
	TypeAuthForceLogout     = "auth.force_logout"     // 强制下线
	TypeAuthUnusualLogin    = "auth.unusual_login"    // 异常登录
	TypeAuthPasswordChanged = "auth.password_changed" // 密码修改
)

// File Service 通知类型
const (
	TypeFileUploadCompleted = "file.upload_completed" // 文件上传完成
	TypeFileProcessing      = "file.processing"       // 文件处理中
	TypeFileExpiring        = "file.expiring"         // 文件即将过期
)

// Push Service 通知类型
const (
	TypePushDeliveryStatus = "push.delivery_status" // 推送发送状态
	TypePushTokenInvalid   = "push.token_invalid"   // 推送Token失效
)

// LiveKit Service 通知类型
const (
	TypeLiveKitCallInvite   = "livekit.call_invite"   // 音视频邀请
	TypeLiveKitCallStatus   = "livekit.call_status"   // 通话状态变更
	TypeLiveKitCallRejected = "livekit.call_rejected" // 通话拒绝
)

// Sync Service 通知类型
const (
	TypeSyncRequest   = "sync.request"   // 同步请求
	TypeSyncCompleted = "sync.completed" // 同步完成
)

// Admin Service 通知类型
const (
	TypeAdminAnnouncement = "admin.announcement" // 系统公告
	TypeAdminUserBanned   = "admin.user_banned"  // 用户封禁
	TypeAdminMaintenance  = "admin.maintenance"  // 系统维护
)

// Conversation Service 通知类型
const (
	TypeConversationUnreadUpdated     = "conversation.unread_updated"      // 未读数更新
	TypeConversationPinUpdated        = "conversation.pin_updated"         // 置顶状态更新
	TypeConversationDeleted           = "conversation.deleted"             // 会话删除
	TypeConversationMuteUpdated       = "conversation.mute_updated"        // 免打扰设置更新
	TypeConversationBurnUpdated       = "conversation.burn_updated"        // 阅后即焚配置变更
	TypeConversationAutoDeleteUpdated = "conversation.auto_delete_updated" // 自动删除配置变更
)
