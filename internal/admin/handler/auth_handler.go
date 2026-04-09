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

// AdminAuthHandler admin authentication handler
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

// Login admin login
// @Summary      admin login
// @Description  admin login with username and password, returns access token
// @Tags         admin-auth
// @Accept       json
// @Produce      json
// @Param        request  body      loginRequest  true  "login info"
// @Success      200      {object}  response.Response{data=object}  "login success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "invalid username or password"
// @Failure      500      {object}  response.Response  "server error"
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

// Logout admin logout
// @Summary      admin logout
// @Tags         admin-auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response  "logout success"
// @Router       /admin/auth/logout [post]
func (h *AdminAuthHandler) Logout(c *gin.Context) {
	response.Success(c, nil)
}

// AdminAuthMiddleware admin JWT verification middleware
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
