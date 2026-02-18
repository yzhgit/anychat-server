package handler

import (
	"github.com/anychat/server/internal/gateway/client"
	gwnotification "github.com/anychat/server/internal/gateway/notification"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/internal/gateway/websocket"
	"github.com/anychat/server/pkg/jwt"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有路由
func RegisterRoutes(r *gin.Engine, clientManager *client.Manager, jwtManager *jwt.Manager,
	wsManager *websocket.Manager, subscriber *gwnotification.Subscriber) {
	// 创建处理器
	authHandler := NewAuthHandler(clientManager)
	userHandler := NewUserHandler(clientManager)
	friendHandler := NewFriendHandler(clientManager)
	groupHandler := NewGroupHandler(clientManager)
	fileHandler := NewFileHandler(clientManager)
	wsHandler := NewWSHandler(clientManager, jwtManager, wsManager, subscriber)
	sessionHandler := NewSessionHandler(clientManager)
	syncHandler := NewSyncHandler(clientManager)
	rtcHandler := NewRTCHandler(clientManager)

	// API v1
	v1 := r.Group("/api/v1")
	{
		// WebSocket接入点（token通过query参数认证）
		v1.GET("/ws", wsHandler.HandleWebSocket)
		// 公开路由（无需认证）
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// 需要认证的路由
		authorized := v1.Group("")
		authorized.Use(gwmiddleware.JWTAuth(jwtManager))
		{
			// Auth路由
			authGroup := authorized.Group("/auth")
			{
				authGroup.POST("/logout", authHandler.Logout)
				authGroup.POST("/password/change", authHandler.ChangePassword)
			}

			// User路由
			users := authorized.Group("/users")
			{
				// 个人资料
				users.GET("/me", userHandler.GetProfile)
				users.PUT("/me", userHandler.UpdateProfile)

				// 用户查询
				users.GET("/:userId", userHandler.GetUserInfo)
				users.GET("/search", userHandler.SearchUsers)

				// 设置
				users.GET("/me/settings", userHandler.GetSettings)
				users.PUT("/me/settings", userHandler.UpdateSettings)

				// 二维码
				users.POST("/me/qrcode/refresh", userHandler.RefreshQRCode)
				users.GET("/qrcode", userHandler.GetUserByQRCode)

				// 推送Token
				users.POST("/me/push-token", userHandler.UpdatePushToken)
			}

			// Friend路由
			friends := authorized.Group("/friends")
			{
				// 好友列表
				friends.GET("", friendHandler.GetFriends)

				// 好友申请
				friends.GET("/requests", friendHandler.GetFriendRequests)
				friends.POST("/requests", friendHandler.SendFriendRequest)
				friends.PUT("/requests/:id", friendHandler.HandleFriendRequest)

				// 好友操作
				friends.DELETE("/:id", friendHandler.DeleteFriend)
				friends.PUT("/:id/remark", friendHandler.UpdateRemark)

				// 黑名单
				friends.GET("/blacklist", friendHandler.GetBlacklist)
				friends.POST("/blacklist", friendHandler.AddToBlacklist)
				friends.DELETE("/blacklist/:id", friendHandler.RemoveFromBlacklist)
			}

			// Group路由
			groups := authorized.Group("/groups")
			{
				// 群组管理
				groups.POST("", groupHandler.CreateGroup)
				groups.GET("", groupHandler.GetMyGroups)
				groups.GET("/:id", groupHandler.GetGroupInfo)
				groups.PUT("/:id", groupHandler.UpdateGroup)
				groups.DELETE("/:id", groupHandler.DissolveGroup)

				// 成员管理
				groups.GET("/:id/members", groupHandler.GetGroupMembers)
				groups.POST("/:id/members", groupHandler.InviteMembers)
				groups.DELETE("/:id/members/:userId", groupHandler.RemoveMember)
				groups.PUT("/:id/members/:userId/role", groupHandler.UpdateMemberRole)
				groups.PUT("/:id/nickname", groupHandler.UpdateMemberNickname)
				groups.POST("/:id/quit", groupHandler.QuitGroup)
				groups.POST("/:id/transfer", groupHandler.TransferOwnership)

				// 入群申请
				groups.POST("/:id/join", groupHandler.JoinGroup)
				groups.GET("/:id/requests", groupHandler.GetJoinRequests)
				groups.PUT("/:id/requests/:requestId", groupHandler.HandleJoinRequest)
			}

			// File路由
			files := authorized.Group("/files")
			{
				files.POST("/upload-token", fileHandler.GenerateUploadToken)
				files.POST("/:fileId/complete", fileHandler.CompleteUpload)
				files.GET("/:fileId/download", fileHandler.GenerateDownloadURL)
				files.GET("/:fileId", fileHandler.GetFileInfo)
				files.DELETE("/:fileId", fileHandler.DeleteFile)
				files.GET("", fileHandler.ListFiles)
			}

			// Session路由
			sessions := authorized.Group("/sessions")
			{
				sessions.GET("", sessionHandler.GetSessions)
				sessions.GET("/unread/total", sessionHandler.GetTotalUnread)
				sessions.GET("/:sessionId", sessionHandler.GetSession)
				sessions.DELETE("/:sessionId", sessionHandler.DeleteSession)
				sessions.PUT("/:sessionId/pin", sessionHandler.SetPinned)
				sessions.PUT("/:sessionId/mute", sessionHandler.SetMuted)
				sessions.POST("/:sessionId/read", sessionHandler.MarkRead)
			}

			// Sync路由
			sync := authorized.Group("/sync")
			{
				sync.POST("", syncHandler.Sync)
				sync.POST("/messages", syncHandler.SyncMessages)
			}

			// RTC路由
			rtc := authorized.Group("/rtc")
			{
				// 一对一通话
				rtc.POST("/calls", rtcHandler.InitiateCall)
				rtc.GET("/calls", rtcHandler.ListCallLogs)
				rtc.GET("/calls/:callId", rtcHandler.GetCallSession)
				rtc.POST("/calls/:callId/join", rtcHandler.JoinCall)
				rtc.POST("/calls/:callId/reject", rtcHandler.RejectCall)
				rtc.POST("/calls/:callId/end", rtcHandler.EndCall)

				// 会议室
				rtc.POST("/meetings", rtcHandler.CreateMeeting)
				rtc.GET("/meetings", rtcHandler.ListMeetings)
				rtc.GET("/meetings/:roomId", rtcHandler.GetMeeting)
				rtc.POST("/meetings/:roomId/join", rtcHandler.JoinMeeting)
				rtc.POST("/meetings/:roomId/end", rtcHandler.EndMeeting)
			}
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "gateway-service",
		})
	})
}
