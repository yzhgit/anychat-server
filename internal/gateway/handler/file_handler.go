package handler

import (
	"strconv"

	filepb "github.com/anychat/server/api/proto/file"
	"github.com/anychat/server/internal/gateway/client"
	gwmiddleware "github.com/anychat/server/internal/gateway/middleware"
	"github.com/anychat/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// FileHandler 文件处理器
type FileHandler struct {
	clientManager *client.Manager
}

// NewFileHandler 创建文件处理器
func NewFileHandler(clientManager *client.Manager) *FileHandler {
	return &FileHandler{
		clientManager: clientManager,
	}
}

// GenerateUploadToken 生成文件上传凭证
// @Summary      生成文件上传凭证
// @Description  生成预签名上传URL，客户端使用此URL直接上传到MinIO
// @Tags         文件
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      object  true  "上传请求"
// @Success      200      {object}  response.Response{data=object}  "成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      500      {object}  response.Response  "服务器错误"
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

// CompleteUpload 完成文件上传
// @Summary      完成文件上传
// @Description  客户端上传完成后，通知服务端激活文件
// @Tags         文件
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        fileId   path      string  true  "文件ID"
// @Success      200      {object}  response.Response{data=object}  "成功"
// @Failure      400      {object}  response.Response  "参数错误"
// @Failure      401      {object}  response.Response  "未授权"
// @Failure      404      {object}  response.Response  "文件不存在"
// @Failure      500      {object}  response.Response  "服务器错误"
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

// GenerateDownloadURL 生成文件下载链接
// @Summary      生成文件下载链接
// @Description  生成预签名下载URL，客户端使用此URL直接从MinIO下载
// @Tags         文件
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        fileId          path   string  true   "文件ID"
// @Param        expiresMinutes  query  int     false  "URL有效期（分钟）" default(60)
// @Success      200             {object}  response.Response{data=object}  "成功"
// @Failure      400             {object}  response.Response  "参数错误"
// @Failure      401             {object}  response.Response  "未授权"
// @Failure      403             {object}  response.Response  "无权访问"
// @Failure      404             {object}  response.Response  "文件不存在"
// @Failure      500             {object}  response.Response  "服务器错误"
// @Router       /files/{fileId}/download [get]
func (h *FileHandler) GenerateDownloadURL(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)
	fileID := c.Param("fileId")

	if fileID == "" {
		response.ParamError(c, "fileId is required")
		return
	}

	// 可选参数：过期时间（分钟）
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

// GetFileInfo 获取文件信息
// @Summary      获取文件信息
// @Description  获取文件元数据信息
// @Tags         文件
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        fileId  path      string  true  "文件ID"
// @Success      200     {object}  response.Response{data=object}  "成功"
// @Failure      400     {object}  response.Response  "参数错误"
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      403     {object}  response.Response  "无权访问"
// @Failure      404     {object}  response.Response  "文件不存在"
// @Failure      500     {object}  response.Response  "服务器错误"
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

// DeleteFile 删除文件
// @Summary      删除文件
// @Description  删除文件（包括MinIO中的对象）
// @Tags         文件
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        fileId  path      string  true  "文件ID"
// @Success      200     {object}  response.Response  "成功"
// @Failure      400     {object}  response.Response  "参数错误"
// @Failure      401     {object}  response.Response  "未授权"
// @Failure      403     {object}  response.Response  "无权访问"
// @Failure      404     {object}  response.Response  "文件不存在"
// @Failure      500     {object}  response.Response  "服务器错误"
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

// ListFiles 列出用户文件
// @Summary      列出用户文件
// @Description  分页列出当前用户的文件，支持按类型过滤
// @Tags         文件
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        fileType  query  string  false  "文件类型（image/video/audio/file）"
// @Param        page      query  int     false  "页码"  default(1)
// @Param        pageSize  query  int     false  "每页数量"  default(20)
// @Success      200       {object}  response.Response{data=object}  "成功"
// @Failure      400       {object}  response.Response  "参数错误"
// @Failure      401       {object}  response.Response  "未授权"
// @Failure      500       {object}  response.Response  "服务器错误"
// @Router       /files [get]
func (h *FileHandler) ListFiles(c *gin.Context) {
	userID := gwmiddleware.GetUserID(c)

	// 解析查询参数
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
