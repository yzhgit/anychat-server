package handler

import (
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	gwnotification "github.com/anychat/server/internal/gateway/notification"
	"github.com/anychat/server/internal/gateway/websocket"
	"github.com/anychat/server/pkg/jwt"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes
func RegisterRoutes(r *gin.Engine, clientManager *client.Manager, jwtManager *jwt.Manager,
	wsManager *websocket.Manager, subscriber *gwnotification.Subscriber) {
	// create handlers
	authHandler := NewAuthHandler(clientManager)
	userHandler := NewUserHandler(clientManager)
	friendHandler := NewFriendHandler(clientManager)
	groupHandler := NewGroupHandler(clientManager)
	fileHandler := NewFileHandler(clientManager)
	logHandler := NewLogHandler(clientManager)
	messageHandler := NewMessageHandler(clientManager)
	wsHandler := NewWSHandler(clientManager, jwtManager, wsManager, subscriber)
	conversationHandler := NewConversationHandler(clientManager)
	syncHandler := NewSyncHandler(clientManager)
	callingHandler := NewCallingHandler(clientManager)
	versionHandler := NewVersionHandler(clientManager)
	// API v1
	v1 := r.Group("/api/v1")
	{
		// WebSocket endpoint (token authenticated via query parameter)
		v1.GET("/ws", wsHandler.HandleWebSocket)
		// public routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/send-code", authHandler.SendCode)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/password/reset", authHandler.ResetPassword)
		}

		// group QR code preview (no auth required)
		v1.GET("/groups/preview", groupHandler.GetGroupPreviewByQRCode)
		v1.GET("/group/preview", groupHandler.GetGroupPreviewByQRCode) // compatible with singular path in design doc

		// routes requiring authentication
		authorized := v1.Group("")
		authorized.Use(gwmiddleware.JWTAuth(jwtManager))
		{
			// Auth routes
			authGroup := authorized.Group("/auth")
			{
				authGroup.POST("/logout", authHandler.Logout)
				authGroup.POST("/password/change", authHandler.ChangePassword)
			}

			// Version routes (client version check - public)
			versions := v1.Group("/versions")
			{
				versions.GET("/check", versionHandler.CheckVersion)
				versions.GET("/latest", versionHandler.GetLatestVersion)
				versions.GET("/list", versionHandler.ListVersions)
			}

			// Version routes requiring auth
			authorizedVersions := authorized.Group("/versions")
			{
				authorizedVersions.POST("/report", versionHandler.ReportVersion)
			}

			// User routes
			users := authorized.Group("/users")
			{
				// personal profile
				users.GET("/me", userHandler.GetProfile)
				users.PUT("/me", userHandler.UpdateProfile)
				users.POST("/me/phone/bind", userHandler.BindPhone)
				users.POST("/me/phone/change", userHandler.ChangePhone)
				users.POST("/me/email/bind", userHandler.BindEmail)
				users.POST("/me/email/change", userHandler.ChangeEmail)

				// user search
				users.GET("/:userId", userHandler.GetUserInfo)
				users.GET("/search", userHandler.SearchUsers)

				// settings
				users.GET("/me/settings", userHandler.GetSettings)
				users.PUT("/me/settings", userHandler.UpdateSettings)

				// QR code
				users.POST("/me/qrcode/refresh", userHandler.RefreshQRCode)
				users.GET("/qrcode", userHandler.GetUserByQRCode)

				// push token
				users.POST("/me/push-token", userHandler.UpdatePushToken)
			}

			// Friend routes
			friends := authorized.Group("/friends")
			{
				// friend list
				friends.GET("", friendHandler.GetFriends)

				// friend requests
				friends.GET("/requests", friendHandler.GetFriendRequests)
				friends.POST("/requests", friendHandler.SendFriendRequest)
				friends.PUT("/requests/:id", friendHandler.HandleFriendRequest)

				// friend operations
				friends.DELETE("/:id", friendHandler.DeleteFriend)
				friends.PUT("/:id/remark", friendHandler.UpdateRemark)

				// blacklist
				friends.GET("/blacklist", friendHandler.GetBlacklist)
				friends.POST("/blacklist", friendHandler.AddToBlacklist)
				friends.DELETE("/blacklist/:id", friendHandler.RemoveFromBlacklist)
			}

			// Group routes
			groups := authorized.Group("/groups")
			{
				// group management
				groups.POST("", groupHandler.CreateGroup)
				groups.GET("", groupHandler.GetMyGroups)
				groups.POST("/join-by-qrcode", groupHandler.JoinGroupByQRCode)
				groups.GET("/:id", groupHandler.GetGroupInfo)
				groups.PUT("/:id", groupHandler.UpdateGroup)
				groups.DELETE("/:id", groupHandler.DissolveGroup)

				// member management
				groups.GET("/:id/members", groupHandler.GetGroupMembers)
				groups.POST("/:id/members", groupHandler.InviteMembers)
				groups.DELETE("/:id/members/:userId", groupHandler.RemoveMember)
				groups.PUT("/:id/members/:userId/mute", groupHandler.MuteMember)
				groups.DELETE("/:id/members/:userId/mute", groupHandler.UnmuteMember)
				groups.PUT("/:id/members/:userId/role", groupHandler.UpdateMemberRole)
				groups.PUT("/:id/nickname", groupHandler.UpdateMemberNickname)
				groups.PUT("/:id/remark", groupHandler.UpdateMemberRemark)
				groups.POST("/:id/quit", groupHandler.QuitGroup)
				groups.POST("/:id/transfer", groupHandler.TransferOwnership)
				groups.GET("/:id/settings", groupHandler.GetGroupSettings)
				groups.PUT("/:id/settings", groupHandler.UpdateGroupSettings)
				groups.PUT("/:id/mute", groupHandler.SetGroupMute)
				groups.POST("/:id/pin", groupHandler.PinGroupMessage)
				groups.DELETE("/:id/pin/:messageId", groupHandler.UnpinGroupMessage)
				groups.GET("/:id/pins", groupHandler.GetPinnedMessages)

				// QR code
				groups.GET("/:id/qrcode", groupHandler.GetGroupQRCode)
				groups.POST("/:id/qrcode/refresh", groupHandler.RefreshGroupQRCode)

				// join requests
				groups.POST("/:id/join", groupHandler.JoinGroup)
				groups.GET("/:id/requests", groupHandler.GetJoinRequests)
				groups.PUT("/:id/requests/:requestId", groupHandler.HandleJoinRequest)
			}

			// compatible with singular path in design doc
			groupAlias := authorized.Group("/group")
			{
				groupAlias.GET("/list", groupHandler.GetMyGroups)
				groupAlias.GET("/:id", groupHandler.GetGroupInfo)
				groupAlias.PUT("/:id/remark", groupHandler.UpdateMemberRemark)
				groupAlias.POST("/join-by-qrcode", groupHandler.JoinGroupByQRCode)
				groupAlias.GET("/:id/qrcode", groupHandler.GetGroupQRCode)
				groupAlias.POST("/:id/qrcode/refresh", groupHandler.RefreshGroupQRCode)
			}

			// File routes
			files := authorized.Group("/files")
			{
				files.POST("/upload-token", fileHandler.GenerateUploadToken)
				files.POST("/:fileId/complete", fileHandler.CompleteUpload)
				files.GET("/:fileId/download", fileHandler.GenerateDownloadURL)
				files.GET("/:fileId", fileHandler.GetFileInfo)
				files.DELETE("/:fileId", fileHandler.DeleteFile)
				files.GET("", fileHandler.ListFiles)
			}

			// Log routes
			logs := authorized.Group("/logs")
			{
				logs.POST("/upload", logHandler.UploadLog)
				logs.POST("/complete", logHandler.CompleteUpload)
				logs.GET("", logHandler.ListLogs)
				logs.GET("/:logId/download", logHandler.DownloadLog)
				logs.DELETE("/:logId", logHandler.DeleteLog)
			}

			// Message routes
			messages := authorized.Group("/messages")
			{
				messages.POST("", messageHandler.SendMessage)
				messages.GET("", messageHandler.GetMessages)
				messages.GET("/search", messageHandler.SearchMessages)
				messages.GET("/:messageId", messageHandler.GetMessageByID)
				messages.POST("/read-triggers", messageHandler.AckReadTriggers)
				messages.POST("/recall", messageHandler.RecallMessage)
				messages.DELETE("/:messageId", messageHandler.DeleteMessage)
			}

			// Conversation routes
			conversations := authorized.Group("/conversations")
			{
				conversations.GET("", conversationHandler.GetConversations)
				conversations.GET("/unread/total", conversationHandler.GetTotalUnread)
				conversations.GET("/:conversationId", conversationHandler.GetConversation)
				conversations.GET("/:conversationId/messages/unread-count", conversationHandler.GetMessageUnreadCount)
				conversations.GET("/:conversationId/messages/read-receipts", conversationHandler.GetMessageReadReceipts)
				conversations.GET("/:conversationId/messages/sequence", conversationHandler.GetMessageSequence)
				conversations.POST("/:conversationId/messages/read", conversationHandler.MarkMessagesRead)
				conversations.DELETE("/:conversationId", conversationHandler.DeleteConversation)
				conversations.PUT("/:conversationId/pin", conversationHandler.SetPinned)
				conversations.PUT("/:conversationId/mute", conversationHandler.SetMuted)
				conversations.PUT("/:conversationId/burn", conversationHandler.SetBurnAfterReading)
				conversations.PUT("/:conversationId/auto_delete", conversationHandler.SetAutoDelete)
				conversations.POST("/:conversationId/read-all", conversationHandler.MarkRead)
			}

			// Sync routes
			sync := authorized.Group("/sync")
			{
				sync.POST("", syncHandler.Sync)
				sync.POST("/messages", syncHandler.SyncMessages)
			}

			// Calling routes (recommended)
			registerCallingRoutes(authorized.Group("/calling"), callingHandler)
		}
	}

	// health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "gateway-service",
		})
	})
}

func registerCallingRoutes(group *gin.RouterGroup, handler *CallingHandler) {
	// one-on-one calls
	group.POST("/calls", handler.InitiateCall)
	group.GET("/calls", handler.ListCallLogs)
	group.GET("/calls/:callId", handler.GetCallSession)
	group.POST("/calls/:callId/join", handler.JoinCall)
	group.POST("/calls/:callId/reject", handler.RejectCall)
	group.POST("/calls/:callId/end", handler.EndCall)

	// meeting rooms
	group.POST("/meetings", handler.CreateMeeting)
	group.GET("/meetings", handler.ListMeetings)
	group.GET("/meetings/:roomId", handler.GetMeeting)
	group.POST("/meetings/:roomId/join", handler.JoinMeeting)
	group.POST("/meetings/:roomId/end", handler.EndMeeting)
}
