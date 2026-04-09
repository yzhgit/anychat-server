package handler

import (
	"strconv"

	filepb "github.com/anychat/server/api/proto/file"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// FileHandler file handler
type FileHandler struct {
	clientManager *client.Manager
}

// NewFileHandler creates file handler
func NewFileHandler(clientManager *client.Manager) *FileHandler {
	return &FileHandler{
		clientManager: clientManager,
	}
}

// GenerateUploadToken generate file upload token
// @Summary      generate file upload token
// @Description  Generate presigned upload URL, client uses this URL to upload directly to MinIO
// @Tags         file
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      object  true  "upload request"
// @Success      200      {object}  response.Response{data=object}  "success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /files/upload-token [post]
func (h *FileHandler) GenerateUploadToken(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	var req struct {
		FileName     string `json:"fileName" binding:"required" example:"photo.jpg"`
		FileSize     int64  `json:"fileSize" binding:"required,gt=0" example:"1024000"`
		MimeType     string `json:"mimeType" binding:"required" example:"image/jpeg"`
		FileType     string `json:"fileType" binding:"required,oneof=image video audio file" example:"image"`
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
		MimeType:     req.MimeType,
		FileType:     req.FileType,
		ExpiresHours: &req.ExpiresHours,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// CompleteUpload complete file upload
// @Summary      complete file upload
// @Description  After client finishes upload, notify server to activate file
// @Tags         file
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        fileId   path      string  true  "file ID"
// @Success      200      {object}  response.Response{data=object}  "success"
// @Failure      400      {object}  response.Response  "parameter error"
// @Failure      401      {object}  response.Response  "unauthorized"
// @Failure      404      {object}  response.Response  "file not found"
// @Failure      500      {object}  response.Response  "server error"
// @Router       /files/{fileId}/complete [post]
func (h *FileHandler) CompleteUpload(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	fileID := c.Param("fileId")

	if fileID == "" {
		response.ParamError(c, "fileId is required")
		return
	}

	resp, err := h.clientManager.File().CompleteUpload(c.Request.Context(), &filepb.CompleteUploadRequest{
		FileId: fileID,
		UserId: userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// GenerateDownloadURL generate file download URL
// @Summary      generate file download URL
// @Description  Generate presigned download URL, client uses this URL to download directly from MinIO
// @Tags         file
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        fileId          path   string  true   "file ID"
// @Param        expiresMinutes  query  int     false  "URL expiration (minutes)" default(60)
// @Success      200             {object}  response.Response{data=object}  "success"
// @Failure      400             {object}  response.Response  "parameter error"
// @Failure      401             {object}  response.Response  "unauthorized"
// @Failure      403             {object}  response.Response  "no permission"
// @Failure      404             {object}  response.Response  "file not found"
// @Failure      500             {object}  response.Response  "server error"
// @Router       /files/{fileId}/download [get]
func (h *FileHandler) GenerateDownloadURL(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	fileID := c.Param("fileId")

	if fileID == "" {
		response.ParamError(c, "fileId is required")
		return
	}

	// Optional parameter: expiration time (minutes)
	var expiresMinutes *int32
	if expiresStr := c.Query("expiresMinutes"); expiresStr != "" {
		if expires, err := strconv.Atoi(expiresStr); err == nil && expires > 0 {
			expiresInt32 := int32(expires)
			expiresMinutes = &expiresInt32
		}
	}

	resp, err := h.clientManager.File().GenerateDownloadURL(c.Request.Context(), &filepb.GenerateDownloadURLRequest{
		FileId:         fileID,
		UserId:         userID,
		ExpiresMinutes: expiresMinutes,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// GetFileInfo get file info
// @Summary      get file info
// @Description  Get file metadata info
// @Tags         file
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        fileId  path      string  true  "file ID"
// @Success      200     {object}  response.Response{data=object}  "success"
// @Failure      400     {object}  response.Response  "parameter error"
// @Failure      401     {object}  response.Response  "unauthorized"
// @Failure      403     {object}  response.Response  "no permission"
// @Failure      404     {object}  response.Response  "file not found"
// @Failure      500     {object}  response.Response  "server error"
// @Router       /files/{fileId} [get]
func (h *FileHandler) GetFileInfo(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	fileID := c.Param("fileId")

	if fileID == "" {
		response.ParamError(c, "fileId is required")
		return
	}

	resp, err := h.clientManager.File().GetFileInfo(c.Request.Context(), &filepb.GetFileInfoRequest{
		FileId: fileID,
		UserId: userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}

// DeleteFile delete file
// @Summary      delete file
// @Description  Delete file (including object in MinIO)
// @Tags         file
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        fileId  path      string  true  "file ID"
// @Success      200     {object}  response.Response  "success"
// @Failure      400     {object}  response.Response  "parameter error"
// @Failure      401     {object}  response.Response  "unauthorized"
// @Failure      403     {object}  response.Response  "no permission"
// @Failure      404     {object}  response.Response  "file not found"
// @Failure      500     {object}  response.Response  "server error"
// @Router       /files/{fileId} [delete]
func (h *FileHandler) DeleteFile(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	fileID := c.Param("fileId")

	if fileID == "" {
		response.ParamError(c, "fileId is required")
		return
	}

	_, err := h.clientManager.File().DeleteFile(c.Request.Context(), &filepb.DeleteFileRequest{
		FileId: fileID,
		UserId: userID,
	})

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, gin.H{"success": true})
}

// ListFiles list user files
// @Summary      list user files
// @Description  Paginate list of current user's files, supports filtering by type
// @Tags         file
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        fileType  query  string  false  "file type (image/video/audio/file)"
// @Param        page      query  int     false  "page number"  default(1)
// @Param        pageSize  query  int     false  "page size"  default(20)
// @Success      200       {object}  response.Response{data=object}  "success"
// @Failure      400       {object}  response.Response  "parameter error"
// @Failure      401       {object}  response.Response  "unauthorized"
// @Failure      500       {object}  response.Response  "server error"
// @Router       /files [get]
func (h *FileHandler) ListFiles(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	// Parse query parameters
	fileType := c.Query("fileType")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	req := &filepb.ListUserFilesRequest{
		UserId:   userID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	}

	if fileType != "" {
		req.FileType = &fileType
	}

	resp, err := h.clientManager.File().ListUserFiles(c.Request.Context(), req)

	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, resp)
}
