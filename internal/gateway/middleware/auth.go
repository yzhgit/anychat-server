package middleware

import (
	"strings"

	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyUserID 用户ID在context中的key
	ContextKeyUserID = "user_id"
	// ContextKeyDeviceID 设备ID在context中的key
	ContextKeyDeviceID = "device_id"
	// ContextKeyDeviceType 设备类型在context中的key
	ContextKeyDeviceType = "device_type"
)

// JWTAuth JWT认证中间件
func JWTAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从header获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, 401, "missing authorization header")
			c.Abort()
			return
		}

		// 解析Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, 401, "invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]

		// 验证token
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			response.Error(c, 401, "invalid or expired token")
			c.Abort()
			return
		}

		// 将用户信息注入context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyDeviceID, claims.DeviceID)
		c.Set(ContextKeyDeviceType, claims.DeviceType)

		c.Next()
	}
}

// GetUserID 从context获取用户ID
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return ""
	}
	return userID.(string)
}

// GetDeviceID 从context获取设备ID
func GetDeviceID(c *gin.Context) string {
	deviceID, exists := c.Get(ContextKeyDeviceID)
	if !exists {
		return ""
	}
	return deviceID.(string)
}

// GetDeviceType 从context获取设备类型
func GetDeviceType(c *gin.Context) string {
	deviceType, exists := c.Get(ContextKeyDeviceType)
	if !exists {
		return ""
	}
	return deviceType.(string)
}
