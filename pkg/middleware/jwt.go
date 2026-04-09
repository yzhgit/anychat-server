package middleware

import (
	"strings"

	"github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	// UserIDKey user ID key in context
	UserIDKey = "userID"
	// DeviceIDKey device ID key in context
	DeviceIDKey = "deviceID"
	// DeviceTypeKey device type key in context
	DeviceTypeKey = "deviceType"
)

// JWTAuth JWT authentication middleware
func JWTAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from request header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "Missing authentication token")
			c.Abort()
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Unauthorized(c, "Invalid token format")
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtManager.ValidateAccessToken(parts[1])
		if err != nil {
			response.Error(c, errors.CodeTokenInvalid, "Invalid token")
			c.Abort()
			return
		}

		// Store user info in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(DeviceIDKey, claims.DeviceID)
		c.Set(DeviceTypeKey, claims.DeviceType)

		c.Next()
	}
}

// GetUserID retrieves user ID from context
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return ""
	}
	return userID.(string)
}

// GetDeviceID retrieves device ID from context
func GetDeviceID(c *gin.Context) string {
	deviceID, exists := c.Get(DeviceIDKey)
	if !exists {
		return ""
	}
	return deviceID.(string)
}

// GetDeviceType retrieves device type from context
func GetDeviceType(c *gin.Context) string {
	deviceType, exists := c.Get(DeviceTypeKey)
	if !exists {
		return ""
	}
	return deviceType.(string)
}
