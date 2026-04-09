package notification

// Define notification type constants for each service

// Friend Service notification types
const (
	TypeFriendRequest        = "friend.request"           // Friend request
	TypeFriendRequestHandled = "friend.request_handled"   // Friend request handled result
	TypeFriendAdded          = "friend.added"             // Automatically added as friend (no verification needed)
	TypeFriendDeleted        = "friend.deleted"           // Friend deleted
	TypeFriendRemarkUpdated  = "friend.remark_updated"    // Friend remark updated
	TypeBlacklistChanged     = "friend.blacklist_changed" // Blacklist changed
)

// Group Service notification types
const (
	TypeGroupInvited         = "group.invited"          // Group invitation
	TypeGroupMemberJoined    = "group.member_joined"    // Member joined
	TypeGroupMemberLeft      = "group.member_left"      // Member left
	TypeGroupMemberMuted     = "group.member_muted"     // Member muted
	TypeGroupMemberUnmuted   = "group.member_unmuted"   // Member unmuted
	TypeGroupInfoUpdated     = "group.info_updated"     // Group info updated
	TypeGroupSettingsUpdated = "group.settings_updated" // Group settings updated
	TypeGroupRoleChanged     = "group.role_changed"     // Role changed
	TypeGroupMuted           = "group.muted"            // Group muted
	TypeGroupMessagePinned   = "group.message_pinned"   // Group message pinned
	TypeGroupMessageUnpinned = "group.message_unpinned" // Group message unpinned
	TypeGroupJoinRequested   = "group.join_requested"   // Join request
	TypeGroupQRCodeRefreshed = "group.qrcode_refreshed" // Group QR code refreshed
	TypeGroupDisbanded       = "group.disbanded"        // Group disbanded
)

// Message Service notification types
const (
	TypeMessageNew         = "message.new"          // New message
	TypeMessageReadReceipt = "message.read_receipt" // Read receipt
	TypeMessageRecalled    = "message.recalled"     // Message recalled
	TypeMessageTyping      = "message.typing"       // Typing
	TypeMessageMentioned   = "message.mentioned"    // Mentioned
	TypeMessageAutoDeleted = "message.auto_deleted" // Message auto deleted
)

// User Service notification types
const (
	TypeUserProfileUpdated       = "user.profile_updated"        // User profile updated
	TypeUserFriendProfileChanged = "user.friend_profile_changed" // Friend profile changed
	TypeUserStatusChanged        = "user.status_changed"         // Online status changed
)

// Auth Service notification types
const (
	TypeAuthForceLogout     = "auth.force_logout"     // Force logout
	TypeAuthUnusualLogin    = "auth.unusual_login"    // Unusual login
	TypeAuthPasswordChanged = "auth.password_changed" // Password changed
)

// File Service notification types
const (
	TypeFileUploadCompleted = "file.upload_completed" // File upload completed
	TypeFileProcessing      = "file.processing"       // File processing
	TypeFileExpiring        = "file.expiring"         // File expiring soon
)

// Push Service notification types
const (
	TypePushDeliveryStatus = "push.delivery_status" // Push delivery status
	TypePushTokenInvalid   = "push.token_invalid"   // Push token invalid
)

// LiveKit Service notification types
const (
	TypeLiveKitCallInvite   = "livekit.call_invite"   // Audio/Video invitation
	TypeLiveKitCallStatus   = "livekit.call_status"   // Call status changed
	TypeLiveKitCallRejected = "livekit.call_rejected" // Call rejected
)

// Sync Service notification types
const (
	TypeSyncRequest   = "sync.request"   // Sync request
	TypeSyncCompleted = "sync.completed" // Sync completed
)

// Admin Service notification types
const (
	TypeAdminAnnouncement = "admin.announcement" // System announcement
	TypeAdminUserBanned   = "admin.user_banned"  // User banned
	TypeAdminMaintenance  = "admin.maintenance"  // System maintenance
)

// Conversation Service notification types
const (
	TypeConversationUnreadUpdated     = "conversation.unread_updated"      // Unread count updated
	TypeConversationPinUpdated        = "conversation.pin_updated"         // Pin status updated
	TypeConversationDeleted           = "conversation.deleted"             // Conversation deleted
	TypeConversationMuteUpdated       = "conversation.mute_updated"        // Do not disturb updated
	TypeConversationBurnUpdated       = "conversation.burn_updated"        // Burn after reading config changed
	TypeConversationAutoDeleteUpdated = "conversation.auto_delete_updated" // Auto delete config changed
)
