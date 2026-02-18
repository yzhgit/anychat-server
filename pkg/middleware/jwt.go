package middleware

import (
	"strings"

	"github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	// UserIDKey 上下文中的用户ID键
	UserIDKey = "userID"
	// DeviceIDKey 上下文中的设备ID键
	DeviceIDKey = "deviceID"
	// DeviceTypeKey 上下文中的设备类型键
	DeviceTypeKey = "deviceType"
)

// JWTAuth JWT认证中间件
func JWTAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "缺少认证Token")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Unauthorized(c, "Token格式错误")
			c.Abort()
			return
		}

		// 验证Token
		claims, err := jwtManager.ValidateAccessToken(parts[1])
		if err != nil {
			response.Error(c, errors.CodeTokenInvalid, "Token无效")
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set(UserIDKey, claims.UserID)
		c.Set(DeviceIDKey, claims.DeviceID)
		c.Set(DeviceTypeKey, claims.DeviceType)

		c.Next()
	}
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return ""
	}
	return userID.(string)
}

// GetDeviceID 从上下文获取设备ID
func GetDeviceID(c *gin.Context) string {
	deviceID, exists := c.Get(DeviceIDKey)
	if !exists {
		return ""
	}
	return deviceID.(string)
}

// GetDeviceType 从上下文获取设备类型
func GetDeviceType(c *gin.Context) string {
	deviceType, exists := c.Get(DeviceTypeKey)
	if !exists {
		return ""
	}
	return deviceType.(string)
}
