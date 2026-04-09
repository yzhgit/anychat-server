package handler

import (
	"strconv"

	filepb "github.com/anychat/server/api/proto/file"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

const fileTypeLog = "log"

type LogHandler struct {
	clientManager *client.Manager
}

func NewLogHandler(clientManager *client.Manager) *LogHandler {
	return &LogHandler{
		clientManager: clientManager,
	}
}

func (h *LogHandler) UploadLog(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req struct {
		FileName     string `json:"fileName" binding:"required" example:"app.log"`
		FileSize     int64  `json:"fileSize" binding:"required,gt=0" example:"1024000"`
		ExpiresHours int32  `json:"expiresHours,omitempty" example:"0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.clientManager.File().GenerateUploadToken(c.Request.Context(), &filepb.GenerateUploadTokenRequest{
		UserId:       userID,
		FileName:     req.FileName,
		FileSize:     req.FileSize,
		MimeType:     "text/plain",
		FileType:     fileTypeLog,
		ExpiresHours: &req.ExpiresHours,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"file_id":    resp.FileId,
		"upload_url": resp.UploadUrl,
		"expires_in": resp.ExpiresIn,
	})
}

func (h *LogHandler) CompleteUpload(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req struct {
		FileID string `json:"file_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, err.Error())
		return
	}

	resp, err := h.clientManager.File().CompleteUpload(c.Request.Context(), &filepb.CompleteUploadRequest{
		FileId: req.FileID,
		UserId: userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

func (h *LogHandler) ListLogs(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	fileType := fileTypeLog
	req := &filepb.ListUserFilesRequest{
		UserId:   userID,
		FileType: &fileType,
		Page:     int32(page),
		PageSize: int32(pageSize),
	}

	resp, err := h.clientManager.File().ListUserFiles(c.Request.Context(), req)

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	var logs []gin.H
	for _, f := range resp.Files {
		logs = append(logs, gin.H{
			"log_id":     f.FileId,
			"file_id":    f.FileId,
			"file_name":  f.FileName,
			"file_size":  f.FileSize,
			"created_at": f.CreatedAt,
		})
	}

	response.Success(c, gin.H{
		"logs":      logs,
		"total":     resp.Total,
		"page":      resp.Page,
		"page_size": resp.PageSize,
	})
}

func (h *LogHandler) DownloadLog(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	logID := c.Param("logId")

	if logID == "" {
		response.ParamError(c, "logId is required")
		return
	}

	fileResp, err := h.clientManager.File().GetFileInfo(c.Request.Context(), &filepb.GetFileInfoRequest{
		FileId: logID,
		UserId: userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	if fileResp.FileType != fileTypeLog {
		response.Forbidden(c, "access denied")
		return
	}

	var expiresMinutes *int32
	if expiresStr := c.Query("expiresMinutes"); expiresStr != "" {
		if expires, err := strconv.Atoi(expiresStr); err == nil && expires > 0 {
			expiresInt32 := int32(expires)
			expiresMinutes = &expiresInt32
		}
	}

	downloadResp, err := h.clientManager.File().GenerateDownloadURL(c.Request.Context(), &filepb.GenerateDownloadURLRequest{
		FileId:         logID,
		UserId:         userID,
		ExpiresMinutes: expiresMinutes,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{
		"download_url": downloadResp.DownloadUrl,
		"expires_in":   downloadResp.ExpiresIn,
	})
}

func (h *LogHandler) DeleteLog(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	logID := c.Param("logId")

	if logID == "" {
		response.ParamError(c, "logId is required")
		return
	}

	fileResp, err := h.clientManager.File().GetFileInfo(c.Request.Context(), &filepb.GetFileInfoRequest{
		FileId: logID,
		UserId: userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	if fileResp.FileType != fileTypeLog {
		response.Forbidden(c, "access denied")
		return
	}

	_, err = h.clientManager.File().DeleteFile(c.Request.Context(), &filepb.DeleteFileRequest{
		FileId: logID,
		UserId: userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{"success": true})
}
