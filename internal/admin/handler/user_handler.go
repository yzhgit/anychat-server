package handler

import (
	"net/http"
	"strconv"

	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// AdminUserMgmtHandler admin user management handler
type AdminUserMgmtHandler struct {
	svc service.AdminService
}

func NewAdminUserMgmtHandler(svc service.AdminService) *AdminUserMgmtHandler {
	return &AdminUserMgmtHandler{svc: svc}
}

// ListUsers query user list
// @Summary      query user list
// @Description  admin search/list system users
// @Tags         admin-user-management
// @Security     BearerAuth
// @Produce      json
// @Param        keyword   query  string  false  "search keyword"
// @Param        page      query  int     false  "page number"
// @Param        pageSize  query  int     false  "page size"
// @Success      200  {object}  response.Response{data=object}  "success"
// @Router       /admin/users [get]
func (h *AdminUserMgmtHandler) ListUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	users, total, err := h.svc.SearchUsers(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, gin.H{"users": users, "total": total, "page": page, "pageSize": pageSize})
}

// GetUser get user details
// @Summary      get user details
// @Tags         admin-user-management
// @Security     BearerAuth
// @Produce      json
// @Param        userId  path  string  true  "user ID"
// @Success      200  {object}  response.Response{data=object}  "success"
// @Router       /admin/users/{userId} [get]
func (h *AdminUserMgmtHandler) GetUser(c *gin.Context) {
	userID := c.Param("userId")
	user, err := h.svc.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	response.Success(c, user)
}

type banUserRequest struct {
	Reason string `json:"reason"`
}

// BanUser ban user
// @Summary      ban user
// @Tags         admin-user-management
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        userId   path  string          true  "user ID"
// @Param        request  body  banUserRequest  false "ban reason"
// @Success      200  {object}  response.Response  "success"
// @Router       /admin/users/{userId}/ban [post]
func (h *AdminUserMgmtHandler) BanUser(c *gin.Context) {
	adminID := getAdminID(c)
	userID := c.Param("userId")
	var req banUserRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.svc.BanUser(c.Request.Context(), adminID, userID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, nil)
}

// UnbanUser unban user
// @Summary      unban user
// @Tags         admin-user-management
// @Security     BearerAuth
// @Produce      json
// @Param        userId  path  string  true  "user ID"
// @Success      200  {object}  response.Response  "success"
// @Router       /admin/users/{userId}/unban [post]
func (h *AdminUserMgmtHandler) UnbanUser(c *gin.Context) {
	adminID := getAdminID(c)
	userID := c.Param("userId")

	if err := h.svc.UnbanUser(c.Request.Context(), adminID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, nil)
}
