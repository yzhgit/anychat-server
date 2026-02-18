package handler

import (
	"net/http"
	"strconv"

	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// AdminStatsHandler 统计数据处理器
type AdminStatsHandler struct {
	svc service.AdminService
}

func NewAdminStatsHandler(svc service.AdminService) *AdminStatsHandler {
	return &AdminStatsHandler{svc: svc}
}

// GetOverview 系统统计概览
// @Summary      系统统计概览
// @Tags         管理后台-统计
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=object}  "成功"
// @Router       /admin/stats/overview [get]
func (h *AdminStatsHandler) GetOverview(c *gin.Context) {
	stats, err := h.svc.GetSystemStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, stats)
}

// AdminAuditHandler 审计日志处理器
type AdminAuditHandler struct {
	svc service.AdminService
}

func NewAdminAuditHandler(svc service.AdminService) *AdminAuditHandler {
	return &AdminAuditHandler{svc: svc}
}

// ListAuditLogs 查询审计日志
// @Summary      查询审计日志
// @Tags         管理后台-审计日志
// @Security     BearerAuth
// @Produce      json
// @Param        adminId   query  string  false  "管理员ID筛选"
// @Param        action    query  string  false  "操作类型筛选"
// @Param        page      query  int     false  "页码"
// @Param        pageSize  query  int     false  "每页数量"
// @Success      200  {object}  response.Response{data=object}  "成功"
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
