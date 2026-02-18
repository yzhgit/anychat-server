package handler

import (
	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册管理后台路由
func RegisterRoutes(r *gin.Engine, svc service.AdminService, jwtManager *jwt.Manager) {
	// 中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// 处理器
	authHandler := NewAdminAuthHandler(svc, jwtManager)
	userHandler := NewAdminUserMgmtHandler(svc)
	groupHandler := NewAdminGroupHandler(svc)
	statsHandler := NewAdminStatsHandler(svc)
	auditHandler := NewAdminAuditHandler(svc)
	configHandler := NewAdminConfigHandler(svc)
	adminMgmtHandler := NewAdminManageHandler(svc)

	api := r.Group("/api/admin")
	{
		// 公开路由（无需认证）
		api.POST("/auth/login", authHandler.Login)

		// 需要管理员认证的路由
		auth := api.Group("")
		auth.Use(AdminAuthMiddleware(jwtManager))
		{
			auth.POST("/auth/logout", authHandler.Logout)

			// 用户管理
			users := auth.Group("/users")
			{
				users.GET("", userHandler.ListUsers)
				users.GET("/:userId", userHandler.GetUser)
				users.POST("/:userId/ban", userHandler.BanUser)
				users.POST("/:userId/unban", userHandler.UnbanUser)
			}

			// 群组管理
			groups := auth.Group("/groups")
			{
				groups.GET("/:groupId", groupHandler.GetGroup)
				groups.DELETE("/:groupId", groupHandler.DissolveGroup)
			}

			// 统计
			stats := auth.Group("/stats")
			{
				stats.GET("/overview", statsHandler.GetOverview)
			}

			// 审计日志
			auth.GET("/audit-logs", auditHandler.ListAuditLogs)

			// 系统配置
			config := auth.Group("/config")
			{
				config.GET("", configHandler.ListConfigs)
				config.PUT("/:key", configHandler.UpdateConfig)
			}

			// 管理员账号管理
			admins := auth.Group("/admins")
			{
				admins.GET("", adminMgmtHandler.ListAdmins)
				admins.POST("", adminMgmtHandler.CreateAdmin)
				admins.PUT("/:adminId/status", adminMgmtHandler.UpdateAdminStatus)
			}
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "admin-service"})
	})
}
