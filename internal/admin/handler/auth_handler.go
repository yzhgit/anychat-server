package handler

import (
	"net/http"
	"strings"

	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

const adminIDKey = "adminID"

// AdminAuthHandler 管理员认证处理器
type AdminAuthHandler struct {
	svc        service.AdminService
	jwtManager *jwt.Manager
}

func NewAdminAuthHandler(svc service.AdminService, jwtManager *jwt.Manager) *AdminAuthHandler {
	return &AdminAuthHandler{svc: svc, jwtManager: jwtManager}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 管理员登录
// @Summary      管理员登录
// @Description  管理员通过用户名密码登录，返回访问Token
// @Tags         管理后台-认证
// @Accept       json
// @Produce      json
// @Param        request  body      loginRequest  true  "登录信息"
// @Success      200      {object}  response.Response{data=object}  "登录成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "账号或密码错误"
// @Failure      500      {object}  response.Response  "服务器错误"
// @Router       /admin/auth/login [post]
func (h *AdminAuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, admin, err := h.svc.Login(c.Request.Context(), req.Username, req.Password, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, gin.H{
		"token":    token,
		"adminId":  admin.ID,
		"username": admin.Username,
		"role":     admin.Role,
	})
}

// Logout 管理员退出
// @Summary      管理员退出
// @Tags         管理后台-认证
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response  "退出成功"
// @Router       /admin/auth/logout [post]
func (h *AdminAuthHandler) Logout(c *gin.Context) {
	response.Success(c, nil)
}

// AdminAuthMiddleware 管理员JWT验证中间件
func AdminAuthMiddleware(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}
		claims, err := jwtManager.ValidateAccessToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}
		c.Set(adminIDKey, claims.UserID) // UserID field holds adminID
		c.Next()
	}
}

func getAdminID(c *gin.Context) string {
	if id, ok := c.Get(adminIDKey); ok {
		return id.(string)
	}
	return ""
}
