package handler

import (
	"net/http"
	"strconv"

	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// AdminStatsHandler statistics handler
type AdminStatsHandler struct {
	svc service.AdminService
}

func NewAdminStatsHandler(svc service.AdminService) *AdminStatsHandler {
	return &AdminStatsHandler{svc: svc}
}

// GetOverview system statistics overview
// @Summary      system statistics overview
// @Tags         admin-statistics
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=object}  "success"
// @Router       /admin/stats/overview [get]
func (h *AdminStatsHandler) GetOverview(c *gin.Context) {
	stats, err := h.svc.GetSystemStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, stats)
}

// AdminAuditHandler audit log handler
type AdminAuditHandler struct {
	svc service.AdminService
}

func NewAdminAuditHandler(svc service.AdminService) *AdminAuditHandler {
	return &AdminAuditHandler{svc: svc}
}

// ListAuditLogs query audit logs
// @Summary      query audit logs
// @Tags         admin-audit-logs
// @Security     BearerAuth
// @Produce      json
// @Param        adminId   query  string  false  "admin ID filter"
// @Param        action    query  string  false  "action filter"
// @Param        page      query  int     false  "page number"
// @Param        pageSize  query  int     false  "page size"
// @Success      200  {object}  response.Response{data=object}  "success"
// @Router       /admin/audit-logs [get]
func (h *AdminAuditHandler) ListAuditLogs(c *gin.Context) {
	adminID := c.Query("adminId")
	action := c.Query("action")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	logs, total, err := h.svc.ListAuditLogs(c.Request.Context(), adminID, action, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, gin.H{"logs": logs, "total": total, "page": page, "pageSize": pageSize})
}
