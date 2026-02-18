package handler

import (
	"net/http"

	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// AdminGroupHandler 群组管理处理器
type AdminGroupHandler struct {
	svc service.AdminService
}

func NewAdminGroupHandler(svc service.AdminService) *AdminGroupHandler {
	return &AdminGroupHandler{svc: svc}
}

// GetGroup 获取群组详情
// @Summary      获取群组详情
// @Tags         管理后台-群组管理
// @Security     BearerAuth
// @Produce      json
// @Param        groupId  path  string  true  "群组ID"
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Router       /admin/groups/{groupId} [get]
func (h *AdminGroupHandler) GetGroup(c *gin.Context) {
	groupID := c.Param("groupId")
	group, err := h.svc.GetGroup(c.Request.Context(), groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	response.Success(c, group)
}

// DissolveGroup 解散群组
// @Summary      解散群组
// @Tags         管理后台-群组管理
// @Security     BearerAuth
// @Produce      json
// @Param        groupId  path  string  true  "群组ID"
// @Success      200  {object}  response.Response  "成功"
// @Router       /admin/groups/{groupId} [delete]
func (h *AdminGroupHandler) DissolveGroup(c *gin.Context) {
	adminID := getAdminID(c)
	groupID := c.Param("groupId")

	if err := h.svc.DissolveGroup(c.Request.Context(), adminID, groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, nil)
}
