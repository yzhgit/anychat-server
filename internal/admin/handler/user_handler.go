package handler

import (
	"net/http"
	"strconv"

	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// AdminUserMgmtHandler 管理员用户管理处理器
type AdminUserMgmtHandler struct {
	svc service.AdminService
}

func NewAdminUserMgmtHandler(svc service.AdminService) *AdminUserMgmtHandler {
	return &AdminUserMgmtHandler{svc: svc}
}

// ListUsers 查询用户列表
// @Summary      查询用户列表
// @Description  管理员搜索/列举系统用户
// @Tags         管理后台-用户管理
// @Security     BearerAuth
// @Produce      json
// @Param        keyword   query  string  false  "搜索关键字"
// @Param        page      query  int     false  "页码"
// @Param        pageSize  query  int     false  "每页数量"
// @Success      200  {object}  response.Response{data=object}  "成功"
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

// GetUser 获取用户详情
// @Summary      获取用户详情
// @Tags         管理后台-用户管理
// @Security     BearerAuth
// @Produce      json
// @Param        userId  path  string  true  "用户ID"
// @Success      200  {object}  response.Response{data=object}  "成功"
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

// BanUser 封禁用户
// @Summary      封禁用户
// @Tags         管理后台-用户管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        userId   path  string          true  "用户ID"
// @Param        request  body  banUserRequest  false "封禁原因"
// @Success      200  {object}  response.Response  "成功"
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

// UnbanUser 解封用户
// @Summary      解封用户
// @Tags         管理后台-用户管理
// @Security     BearerAuth
// @Produce      json
// @Param        userId  path  string  true  "用户ID"
// @Success      200  {object}  response.Response  "成功"
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
