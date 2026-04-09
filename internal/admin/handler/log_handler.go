package handler

import (
	"strconv"

	filepb "github.com/anychat/server/api/proto/file"
	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	svc service.AdminService
}

func NewLogHandler(svc service.AdminService) *LogHandler {
	return &LogHandler{svc: svc}
}

func (h *LogHandler) ListLogs(c *gin.Context) {
	userID := c.Query("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var files []*filepb.FileInfo
	var total int64
	var err error

	if userID != "" {
		files, total, err = h.svc.ListLogFiles(c.Request.Context(), userID, page, pageSize)
	} else {
		response.ParamError(c, "user_id is required")
		return
	}

	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}

	var logs []gin.H
	for _, f := range files {
		logs = append(logs, gin.H{
			"log_id":     f.FileId,
			"file_id":    f.FileId,
			"file_name":  f.FileName,
			"file_size":  f.FileSize,
			"user_id":    f.UserId,
			"created_at": f.CreatedAt,
		})
	}

	response.Success(c, gin.H{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *LogHandler) DownloadLog(c *gin.Context) {
	logID := c.Param("logId")
	if logID == "" {
		response.ParamError(c, "logId is required")
		return
	}

	expiresMinutes := int32(60)
	if expiresStr := c.Query("expiresMinutes"); expiresStr != "" {
		if expires, err := strconv.Atoi(expiresStr); err == nil && expires > 0 {
			expiresMinutes = int32(expires)
		}
	}

	downloadURL, expiresIn, err := h.svc.GetLogDownloadURL(c.Request.Context(), logID, expiresMinutes)
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}

	response.Success(c, gin.H{
		"download_url": downloadURL,
		"expires_in":   expiresIn,
	})
}
