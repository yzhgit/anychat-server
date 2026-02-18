package middleware

import (
	"net/http"

	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery 恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, response.Response{
					Code:    http.StatusInternalServerError,
					Message: "Internal Server Error",
					Data:    nil,
				})
			}
		}()
		c.Next()
	}
}
