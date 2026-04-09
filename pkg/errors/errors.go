package errors

import "errors"

// Common error codes
const (
	CodeSuccess       = 0
	CodeParamError    = 1   // Parameter error
	CodeInternalError = 2   // Internal error
	CodeUnauthorized  = 401 // Unauthorized
	CodeForbidden     = 403 // Forbidden
	CodeNotFound      = 404 // Resource not found
)

// Auth Service error codes (10xxx)
const (
	CodeUserExists          = 10101 // User already exists
	CodePasswordWeak        = 10103 // Password too weak
	CodeUserNotFound        = 10104 // User not found
	CodePasswordError       = 10105 // Incorrect password
	CodeAccountDisabled     = 10106 // Account disabled
	CodeRefreshTokenInvalid = 10107 // Invalid RefreshToken
	CodeRefreshTokenExpired = 10108 // RefreshToken expired
	CodeTokenInvalid        = 10109 // Invalid Token
	CodeTokenExpired        = 10110 // Token expired

	// Verification code sub-domain error codes (102xx)
	CodeSendRateLimited        = 10201 // Sending too frequently
	CodeSendLimitReached       = 10202 // Verification code send limit reached
	CodeTargetFormatInvalid    = 10203 // Invalid target format
	CodeSMSServiceError        = 10204 // SMS service error
	CodeEmailServiceError      = 10205 // Email service error
	CodeVerifyCodeError        = 10206 // Incorrect verification code
	CodeVerifyCodeExpired      = 10207 // Verification code expired
	CodeVerifyCodeAlreadyUsed  = 10208 // Verification code already used, please get a new one
	CodeVerifyCodeNotFound     = 10209 // Verification code not found
	CodeVerifyAttemptsExceeded = 10210 // Too many verification attempts
)

// User Service error codes (20xxx)
const (
	CodeNicknameUsed        = 20101 // Nickname already used
	CodeNicknameSensitive   = 20102 // Nickname contains sensitive words
	CodeUserProfileNotFound = 20103 // User not found
	CodeQRCodeExpired       = 20104 // QR code expired
	CodeQRCodeInvalid       = 20105 // Invalid QR code
	CodePhoneFormatInvalid  = 20106 // Invalid phone number format
	CodePhoneAlreadyBound   = 20107 // Phone number already bound
	CodeEmailFormatInvalid  = 20108 // Invalid email format
	CodeEmailAlreadyBound   = 20109 // Email already bound
	CodeOldPhoneNotMatch    = 20110 // Old phone number does not match
	CodeOldEmailNotMatch    = 20111 // Old email does not match
)

// Friend Service error codes (30xxx)
const (
	CodeAlreadyFriend         = 30101 // Already friends
	CodeBlockedByUser         = 30102 // Blocked by user
	CodeDuplicateRequest      = 30103 // Duplicate request
	CodeFriendNotFound        = 30104 // Friend not found
	CodeRequestNotFound       = 30105 // Request not found
	CodeCannotAddSelf         = 30106 // Cannot add yourself as friend
	CodeRequestProcessed      = 30107 // Request already processed
	CodeRequestExpired        = 30108 // Request expired
	CodeFriendLimitReached    = 30109 // Friend limit reached
	CodeTargetFriendLimit     = 30110 // Target friend limit reached
	CodeBlacklistLimitReached = 30111 // Blacklist limit reached
	CodeAlreadyInBlacklist    = 30112 // Already in blacklist
	CodeNotInBlacklist        = 30113 // Not in blacklist
	CodeUserBlocked           = 30114 // User blocked
	CodeRequestExists         = 30115 // Request already exists
	CodePermissionDenied      = 30116 // Permission denied
	CodeNotFriend             = 30117 // Not a friend
)

// Group Service error codes (40xxx)
const (
	CodeGroupNotFound            = 40101 // Group not found
	CodeGroupDissolved           = 40102 // Group dissolved
	CodeGroupMemberTooFew        = 40103 // Insufficient group members
	CodeGroupMemberLimitReached  = 40104 // Group member limit reached
	CodeNotGroupMember           = 40105 // Not a group member
	CodeAlreadyGroupMember       = 40106 // Already a group member
	CodeNoOwnerPermission        = 40107 // No owner permission
	CodeNoAdminPermission        = 40108 // No admin permission
	CodeCannotRemoveOwner        = 40109 // Cannot remove owner
	CodeCannotRemoveAdmin        = 40110 // Cannot remove admin
	CodeGroupNameSensitive       = 40111 // Group name contains sensitive words
	CodeAnnouncementSensitive    = 40112 // Group announcement contains sensitive words
	CodeJoinRequestNotFound      = 40113 // Join request not found
	CodeJoinRequestProcessed     = 40114 // Join request already processed
	CodeMemberMuted              = 40115 // Member muted
	CodeCannotQuitOwnGroup       = 40116 // Cannot quit your own group
	CodeGroupQRExpired           = 40117 // Group QR code expired
	CodeGroupQRInvalid           = 40118 // Invalid group QR code
	CodeGroupPinnedLimitExceeded = 40119 // Group pinned limit exceeded
)

// Message Service error codes (50xxx)
const (
	CodeMessageNotFound         = 50101 // Message not found
	CodeMessageSendFailed       = 50102 // Message send failed
	CodeMessageRecallFailed     = 50103 // Message recall failed
	CodeMessageRecallTimeLimit  = 50104 // Message recall time limit exceeded
	CodeMessageDeleteFailed     = 50105 // Message delete failed
	CodeMessagePermissionDenied = 50106 // Message permission denied
	CodeConversationNotFound    = 50107 // Conversation not found
	CodeSequenceGenerateFailed  = 50108 // Sequence number generation failed
	CodeMarkReadFailed          = 50109 // Mark read failed
	CodeGetUnreadCountFailed    = 50110 // Get unread count failed
	CodeSearchMessageFailed     = 50111 // Search message failed
	CodeInvalidOperation        = 50112 // Invalid operation
	CodeMessageNotInGroup       = 50113 // Message not in this group
)

// File Service error codes (70xxx)
const (
	CodeFileNotFound         = 70101 // File not found
	CodeFileAccessDenied     = 70102 // File access denied
	CodeFileSizeExceeded     = 70103 // File size exceeded
	CodeFileTypeNotAllowed   = 70104 // File type not allowed
	CodeFileUploadFailed     = 70105 // File upload failed
	CodeFileAlreadyExists    = 70106 // File already exists
	CodeInvalidFileID        = 70107 // Invalid file ID
	CodeFileExpired          = 70108 // File expired
	CodeStorageQuotaExceeded = 70109 // Storage quota exceeded
	CodeThumbnailGenFailed   = 70110 // Thumbnail generation failed
)

// Sync Service error codes (11xxx)
const (
	CodeSyncFailed         = 11101 // Sync failed
	CodeSyncMessagesFailed = 11102 // Message sync failed
)

// Version Service error codes (82xxx)
const (
	CodeVersionFormatError   = 82001 // Version format error
	CodeVersionAlreadyExists = 82002 // Version already exists
	CodeVersionNotFound      = 82003 // Version not found
	CodeDownloadUrlInvalid   = 82004 // Invalid download URL
	CodePlatformNotSupported = 82005 // Platform not supported
)

// Push Service error codes (80xxx)
const (
	CodePushFailed        = 80101 // Push failed
	CodePushTokenNotFound = 80102 // Push token not found
	CodePushConfigInvalid = 80103 // Push config invalid
)

// Calling Service error codes (90xxx)
const (
	CodeCallNotFound         = 90101 // Call not found
	CodeCallAlreadyActive    = 90102 // Call already active
	CodeCallPermissionDenied = 90103 // Call permission denied
	CodeCallInvalidStatus    = 90104 // Invalid call status
	CodeMeetingNotFound      = 90105 // Meeting room not found
	CodeMeetingPasswordWrong = 90106 // Meeting room password incorrect
	CodeMeetingAlreadyEnded  = 90107 // Meeting room already ended
	CodeMeetingPermission    = 90108 // Meeting room permission denied
	CodeLiveKitTokenFailed   = 90109 // LiveKit token generation failed
	CodeLiveKitRoomFailed    = 90110 // LiveKit room operation failed
)

// Admin Service error codes (12xxx)
const (
	CodeAdminNotFound        = 12101 // Admin not found
	CodeAdminDisabled        = 12102 // Admin account disabled
	CodeAdminUsernameExists  = 12103 // Admin username already exists
	CodeAdminInvalidPassword = 12104 // Admin password incorrect
	CodeAdminTokenInvalid    = 12105 // Admin token invalid
	CodeAdminPermission      = 12106 // Admin permission denied
	CodeConfigKeyNotFound    = 12107 // Config key not found
)

// Session Service error codes (60xxx)
const (
	CodeSessionNotFound     = 60101 // Session not found
	CodeSessionDeleted      = 60102 // Session deleted
	CodeSessionCreateFailed = 60103 // Session creation failed
	CodeUnreadCountFailed   = 60104 // Unread count error
)

// Error message mapping
var errorMessages = map[int]string{
	CodeSuccess:       "Success",
	CodeParamError:    "Parameter error",
	CodeInternalError: "Internal error",
	CodeUnauthorized:  "Unauthorized",
	CodeForbidden:     "Forbidden",
	CodeNotFound:      "Resource not found",

	CodeUserExists:          "User already exists",
	CodePasswordWeak:        "Password too weak",
	CodeUserNotFound:        "User not found",
	CodePasswordError:       "Incorrect password",
	CodeAccountDisabled:     "Account disabled",
	CodeRefreshTokenInvalid: "Invalid RefreshToken",
	CodeRefreshTokenExpired: "RefreshToken expired",
	CodeTokenInvalid:        "Invalid Token",
	CodeTokenExpired:        "Token expired",

	CodeNicknameUsed:        "Nickname already used",
	CodeNicknameSensitive:   "Nickname contains sensitive words",
	CodeUserProfileNotFound: "User not found",
	CodeQRCodeExpired:       "QR code expired",
	CodeQRCodeInvalid:       "Invalid QR code",
	CodePhoneFormatInvalid:  "Invalid phone number format",
	CodePhoneAlreadyBound:   "Phone number already bound",
	CodeEmailFormatInvalid:  "Invalid email format",
	CodeEmailAlreadyBound:   "Email already bound",
	CodeOldPhoneNotMatch:    "Old phone number does not match",
	CodeOldEmailNotMatch:    "Old email does not match",

	CodeAlreadyFriend:         "Already friends",
	CodeBlockedByUser:         "Blocked by user",
	CodeDuplicateRequest:      "Duplicate request",
	CodeFriendNotFound:        "Friend not found",
	CodeRequestNotFound:       "Request not found",
	CodeCannotAddSelf:         "Cannot add yourself as friend",
	CodeRequestProcessed:      "Request already processed",
	CodeRequestExpired:        "Request expired",
	CodeFriendLimitReached:    "Friend limit reached",
	CodeTargetFriendLimit:     "Target friend limit reached",
	CodeBlacklistLimitReached: "Blacklist limit reached",
	CodeAlreadyInBlacklist:    "Already in blacklist",
	CodeNotInBlacklist:        "Not in blacklist",
	CodeUserBlocked:           "User blocked",
	CodeRequestExists:         "Request already exists",
	CodePermissionDenied:      "Permission denied",
	CodeNotFriend:             "Not a friend",

	CodeGroupNotFound:            "Group not found",
	CodeGroupDissolved:           "Group dissolved",
	CodeGroupMemberTooFew:        "Insufficient group members",
	CodeGroupMemberLimitReached:  "Group member limit reached",
	CodeNotGroupMember:           "Not a group member",
	CodeAlreadyGroupMember:       "Already a group member",
	CodeNoOwnerPermission:        "No owner permission",
	CodeNoAdminPermission:        "No admin permission",
	CodeCannotRemoveOwner:        "Cannot remove owner",
	CodeCannotRemoveAdmin:        "Cannot remove admin",
	CodeGroupNameSensitive:       "Group name contains sensitive words",
	CodeAnnouncementSensitive:    "Group announcement contains sensitive words",
	CodeJoinRequestNotFound:      "Join request not found",
	CodeJoinRequestProcessed:     "Join request already processed",
	CodeMemberMuted:              "Member muted",
	CodeCannotQuitOwnGroup:       "Cannot quit your own group",
	CodeGroupQRExpired:           "Group QR code expired",
	CodeGroupQRInvalid:           "Invalid group QR code",
	CodeGroupPinnedLimitExceeded: "Group pinned limit exceeded, please unpin some first",

	CodeMessageNotFound:         "Message not found",
	CodeMessageSendFailed:       "Message send failed",
	CodeMessageRecallFailed:     "Message recall failed",
	CodeMessageRecallTimeLimit:  "Message recall time limit exceeded",
	CodeMessageDeleteFailed:     "Message delete failed",
	CodeMessagePermissionDenied: "Message permission denied",
	CodeConversationNotFound:    "Conversation not found",
	CodeSequenceGenerateFailed:  "Sequence number generation failed",
	CodeMarkReadFailed:          "Mark read failed",
	CodeGetUnreadCountFailed:    "Get unread count failed",
	CodeSearchMessageFailed:     "Search message failed",
	CodeInvalidOperation:        "Invalid operation",
	CodeMessageNotInGroup:       "Message not in this group",

	CodeFileNotFound:         "File not found",
	CodeFileAccessDenied:     "File access denied",
	CodeFileSizeExceeded:     "File size exceeded",
	CodeFileTypeNotAllowed:   "File type not allowed",
	CodeFileUploadFailed:     "File upload failed",
	CodeFileAlreadyExists:    "File already exists",
	CodeInvalidFileID:        "Invalid file ID",
	CodeFileExpired:          "File expired",
	CodeStorageQuotaExceeded: "Storage quota exceeded",
	CodeThumbnailGenFailed:   "Thumbnail generation failed",

	CodeSessionNotFound:     "Session not found",
	CodeSessionDeleted:      "Session deleted",
	CodeSessionCreateFailed: "Session creation failed",
	CodeUnreadCountFailed:   "Unread count error",

	CodeSendRateLimited:        "Sending too frequently, please try again later",
	CodeSendLimitReached:       "Verification code send limit reached",
	CodeTargetFormatInvalid:    "Invalid target format",
	CodeSMSServiceError:        "SMS service error, please try again later",
	CodeEmailServiceError:      "Email service error, please try again later",
	CodeVerifyCodeError:        "Incorrect verification code",
	CodeVerifyCodeExpired:      "Verification code expired",
	CodeVerifyCodeAlreadyUsed:  "Verification code already used, please get a new one",
	CodeVerifyCodeNotFound:     "Verification code not found",
	CodeVerifyAttemptsExceeded: "Too many verification attempts, please get a new verification code",

	CodeVersionFormatError:   "Version format error",
	CodeVersionAlreadyExists: "Version already exists",
	CodeVersionNotFound:      "Version not found",
	CodeDownloadUrlInvalid:   "Invalid download URL",
	CodePlatformNotSupported: "Platform not supported",
}

// GetMessage returns the error message for a given code
func GetMessage(code int) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	}
	return "Unknown error"
}

// Business represents a business error
type Business struct {
	Code    int
	Message string
}

func (e *Business) Error() string {
	return e.Message
}

// NewBusiness creates a new business error
func NewBusiness(code int, message string) error {
	if message == "" {
		message = GetMessage(code)
	}
	return &Business{
		Code:    code,
		Message: message,
	}
}

// IsBusiness checks if the error is a business error
func IsBusiness(err error) bool {
	var bizErr *Business
	return errors.As(err, &bizErr)
}

// GetBusinessCode retrieves the business error code
func GetBusinessCode(err error) int {
	var bizErr *Business
	if errors.As(err, &bizErr) {
		return bizErr.Code
	}
	return CodeInternalError
}
