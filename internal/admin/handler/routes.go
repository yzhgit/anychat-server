package handler

import (
	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers admin dashboard routes
func RegisterRoutes(r *gin.Engine, svc service.AdminService, jwtManager *jwt.Manager) {
	// Middleware
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Handlers
	authHandler := NewAdminAuthHandler(svc, jwtManager)
	userHandler := NewAdminUserMgmtHandler(svc)
	groupHandler := NewAdminGroupHandler(svc)
	statsHandler := NewAdminStatsHandler(svc)
	auditHandler := NewAdminAuditHandler(svc)
	configHandler := NewAdminConfigHandler(svc)
	adminMgmtHandler := NewAdminManageHandler(svc)
	logHandler := NewLogHandler(svc)

	api := r.Group("/api/admin")
	{
		// Public routes (no authentication required)
		api.POST("/auth/login", authHandler.Login)

		// Routes requiring admin authentication
		auth := api.Group("")
		auth.Use(AdminAuthMiddleware(jwtManager))
		{
			auth.POST("/auth/logout", authHandler.Logout)

			// User management
			users := auth.Group("/users")
			{
				users.GET("", userHandler.ListUsers)
				users.GET("/:userId", userHandler.GetUser)
				users.POST("/:userId/ban", userHandler.BanUser)
				users.POST("/:userId/unban", userHandler.UnbanUser)
			}

			// Group management
			groups := auth.Group("/groups")
			{
				groups.GET("/:groupId", groupHandler.GetGroup)
				groups.DELETE("/:groupId", groupHandler.DissolveGroup)
			}

			// Statistics
			stats := auth.Group("/stats")
			{
				stats.GET("/overview", statsHandler.GetOverview)
			}

			// Audit logs
			auth.GET("/audit-logs", auditHandler.ListAuditLogs)

			// System config
			config := auth.Group("/config")
			{
				config.GET("", configHandler.ListConfigs)
				config.PUT("/:key", configHandler.UpdateConfig)
			}

			// Admin account management
			admins := auth.Group("/admins")
			{
				admins.GET("", adminMgmtHandler.ListAdmins)
				admins.POST("", adminMgmtHandler.CreateAdmin)
				admins.PUT("/:adminId/status", adminMgmtHandler.UpdateAdminStatus)
			}

			// Client log management
			logs := auth.Group("/logs")
			{
				logs.GET("", logHandler.ListLogs)
				logs.GET("/:logId/download", logHandler.DownloadLog)
			}
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "admin-service"})
	})
}
