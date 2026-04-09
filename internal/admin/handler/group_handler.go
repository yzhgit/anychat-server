package handler

import (
	"net/http"

	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// AdminGroupHandler group management handler
type AdminGroupHandler struct {
	svc service.AdminService
}

func NewAdminGroupHandler(svc service.AdminService) *AdminGroupHandler {
	return &AdminGroupHandler{svc: svc}
}

// GetGroup get group details
// @Summary      get group details
// @Tags         admin-group-management
// @Security     BearerAuth
// @Produce      json
// @Param        groupId  path  string  true  "group ID"
// @Success      200  {object}  response.Response{data=object}  "success"
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

// DissolveGroup dissolve group
// @Summary      dissolve group
// @Tags         admin-group-management
// @Security     BearerAuth
// @Produce      json
// @Param        groupId  path  string  true  "group ID"
// @Success      200  {object}  response.Response  "success"
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
