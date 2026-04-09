package middleware

import (
	"strings"

	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyUserID user ID key in context
	ContextKeyUserID = "user_id"
	// ContextKeyDeviceID device ID key in context
	ContextKeyDeviceID = "device_id"
	// ContextKeyDeviceType device type key in context
	ContextKeyDeviceType = "device_type"
)

// JWTAuth JWT authentication middleware
func JWTAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, 401, "missing authorization header")
			c.Abort()
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, 401, "invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			response.Error(c, 401, "invalid or expired token")
			c.Abort()
			return
		}

		// Inject user info into context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyDeviceID, claims.DeviceID)
		c.Set(ContextKeyDeviceType, claims.DeviceType)

		c.Next()
	}
}

// GetUserID get user ID from context
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return ""
	}
	return userID.(string)
}

// GetDeviceID get device ID from context
func GetDeviceID(c *gin.Context) string {
	deviceID, exists := c.Get(ContextKeyDeviceID)
	if !exists {
		return ""
	}
	return deviceID.(string)
}

// GetDeviceType get device type from context
func GetDeviceType(c *gin.Context) string {
	deviceType, exists := c.Get(ContextKeyDeviceType)
	if !exists {
		return ""
	}
	return deviceType.(string)
}
