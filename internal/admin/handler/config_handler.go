package handler

import (
	"net/http"

	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// AdminConfigHandler 系统配置处理器
type AdminConfigHandler struct {
	svc service.AdminService
}

func NewAdminConfigHandler(svc service.AdminService) *AdminConfigHandler {
	return &AdminConfigHandler{svc: svc}
}

// ListConfigs 获取所有系统配置
// @Summary      获取系统配置
// @Tags         管理后台-系统配置
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Router       /admin/config [get]
func (h *AdminConfigHandler) ListConfigs(c *gin.Context) {
	configs, err := h.svc.GetAllConfigs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, configs)
}

type updateConfigRequest struct {
	Value string `json:"value" binding:"required"`
}

// UpdateConfig 更新系统配置
// @Summary      更新系统配置
// @Tags         管理后台-系统配置
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        key      path  string              true  "配置键"
// @Param        request  body  updateConfigRequest true  "配置值"
// @Success      200  {object}  response.Response  "成功"
// @Router       /admin/config/{key} [put]
func (h *AdminConfigHandler) UpdateConfig(c *gin.Context) {
	adminID := getAdminID(c)
	key := c.Param("key")

	var req updateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateConfig(c.Request.Context(), adminID, key, req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, nil)
}

// AdminManageHandler 管理员账号管理处理器
type AdminManageHandler struct {
	svc service.AdminService
}

func NewAdminManageHandler(svc service.AdminService) *AdminManageHandler {
	return &AdminManageHandler{svc: svc}
}

// ListAdmins 查询管理员列表
// @Summary      管理员列表
// @Tags         管理后台-管理员管理
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Router       /admin/admins [get]
func (h *AdminManageHandler) ListAdmins(c *gin.Context) {
	admins, total, err := h.svc.ListAdmins(c.Request.Context(), 1, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	result := make([]gin.H, 0, len(admins))
	for _, a := range admins {
		result = append(result, gin.H{
			"id":          a.ID,
			"username":    a.Username,
			"email":       a.Email,
			"role":        a.Role,
			"status":      a.Status,
			"lastLoginAt": a.LastLoginAt,
			"createdAt":   a.CreatedAt,
		})
	}
	response.Success(c, gin.H{"admins": result, "total": total})
}

type createAdminRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

// CreateAdmin 创建管理员
// @Summary      创建管理员
// @Tags         管理后台-管理员管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body  createAdminRequest  true  "管理员信息"
// @Success      200  {object}  response.Response  "成功"
// @Router       /admin/admins [post]
func (h *AdminManageHandler) CreateAdmin(c *gin.Context) {
	var req createAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Role == "" {
		req.Role = "admin"
	}
	admin, err := h.svc.CreateAdmin(c.Request.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, gin.H{"id": admin.ID, "username": admin.Username, "role": admin.Role})
}

type updateAdminStatusRequest struct {
	Status int8 `json:"status" binding:"required"`
}

// UpdateAdminStatus 启用/禁用管理员
// @Summary      更新管理员状态
// @Tags         管理后台-管理员管理
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        adminId  path  string                    true  "管理员ID"
// @Param        request  body  updateAdminStatusRequest  true  "状态"
// @Success      200  {object}  response.Response  "成功"
// @Router       /admin/admins/{adminId}/status [put]
func (h *AdminManageHandler) UpdateAdminStatus(c *gin.Context) {
	id := c.Param("adminId")
	var req updateAdminStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateAdminStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, nil)
}
