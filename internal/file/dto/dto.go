package dto

// GenerateUploadTokenRequest 生成上传凭证请求
type GenerateUploadTokenRequest struct {
	FileName     string `json:"fileName" binding:"required" example:"photo.jpg"`
	FileSize     int64  `json:"fileSize" binding:"required,gt=0" example:"1024000"`
	MimeType     string `json:"mimeType" binding:"required" example:"image/jpeg"`
	FileType     string `json:"fileType" binding:"required,oneof=image video audio file" example:"image"`
	ExpiresHours *int32 `json:"expiresHours,omitempty" example:"0"`
}

// GenerateUploadTokenResponse 生成上传凭证响应
type GenerateUploadTokenResponse struct {
	FileID    string `json:"fileId" example:"file-123"`
	UploadURL string `json:"uploadUrl" example:"https://minio:9000/..."`
	ExpiresIn int64  `json:"expiresIn" example:"3600"`
}

// CompleteUploadRequest 完成上传请求
type CompleteUploadRequest struct {
	FileID string `json:"fileId" binding:"required" example:"file-123"`
}

// FileInfoResponse 文件信息响应
type FileInfoResponse struct {
	FileID        string            `json:"fileId" example:"file-123"`
	UserID        string            `json:"userId" example:"user-123"`
	FileName      string            `json:"fileName" example:"photo.jpg"`
	FileType      string            `json:"fileType" example:"image"`
	FileSize      int64             `json:"fileSize" example:"1024000"`
	MimeType      string            `json:"mimeType" example:"image/jpeg"`
	StoragePath   string            `json:"storagePath" example:"chat-file/user-123/2024-01-15/uuid.jpg"`
	ThumbnailPath string            `json:"thumbnailPath,omitempty" example:"chat-file/user-123/2024-01-15/uuid_thumb.jpg"`
	BucketName    string            `json:"bucketName" example:"chat-file"`
	Status        int32             `json:"status" example:"1"`
	CreatedAt     int64             `json:"createdAt" example:"1705315200"`
	ExpiresAt     *int64            `json:"expiresAt,omitempty" example:"1705401600"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	DownloadURL   string            `json:"downloadUrl,omitempty" example:"https://minio:9000/..."`
	ThumbnailURL  string            `json:"thumbnailUrl,omitempty" example:"https://minio:9000/..."`
}

// GenerateDownloadURLRequest 生成下载链接请求
type GenerateDownloadURLRequest struct {
	FileID         string `json:"fileId" binding:"required" example:"file-123"`
	ExpiresMinutes *int32 `json:"expiresMinutes,omitempty" example:"60"`
}

// GenerateDownloadURLResponse 生成下载链接响应
type GenerateDownloadURLResponse struct {
	DownloadURL  string `json:"downloadUrl" example:"https://minio:9000/..."`
	ExpiresIn    int64  `json:"expiresIn" example:"3600"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty" example:"https://minio:9000/..."`
}

// ListFilesRequest 列出文件请求
type ListFilesRequest struct {
	FileType *string `form:"fileType" example:"image"`
	Page     int     `form:"page" binding:"required,min=1" example:"1"`
	PageSize int     `form:"pageSize" binding:"required,min=1,max=100" example:"20"`
}

// ListFilesResponse 列出文件响应
type ListFilesResponse struct {
	Files    []*FileInfoResponse `json:"files"`
	Total    int64               `json:"total" example:"100"`
	Page     int                 `json:"page" example:"1"`
	PageSize int                 `json:"pageSize" example:"20"`
}

// DeleteFileRequest 删除文件请求（仅用于内部）
type DeleteFileRequest struct {
	FileID string `json:"fileId" binding:"required" example:"file-123"`
}

// DeleteFileResponse 删除文件响应
type DeleteFileResponse struct {
	Success bool `json:"success" example:"true"`
}
