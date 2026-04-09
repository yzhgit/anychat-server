package dto

// GenerateUploadTokenRequest generate upload token request
type GenerateUploadTokenRequest struct {
	FileName     string `json:"fileName" binding:"required" example:"photo.jpg"`
	FileSize     int64  `json:"fileSize" binding:"required,gt=0" example:"1024000"`
	MimeType     string `json:"mimeType" binding:"required" example:"image/jpeg"`
	FileType     string `json:"fileType" binding:"required,oneof=image video audio file" example:"image"`
	ExpiresHours *int32 `json:"expiresHours,omitempty" example:"0"`
}

// GenerateUploadTokenResponse generate upload token response
type GenerateUploadTokenResponse struct {
	FileID    string `json:"fileId" example:"file-123"`
	UploadURL string `json:"uploadUrl" example:"https://minio:9000/..."`
	ExpiresIn int64  `json:"expiresIn" example:"3600"`
}

// CompleteUploadRequest complete upload request
type CompleteUploadRequest struct {
	FileID string `json:"fileId" binding:"required" example:"file-123"`
}

// FileInfoResponse file info response
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

// GenerateDownloadURLRequest generate download URL request
type GenerateDownloadURLRequest struct {
	FileID         string `json:"fileId" binding:"required" example:"file-123"`
	ExpiresMinutes *int32 `json:"expiresMinutes,omitempty" example:"60"`
}

// GenerateDownloadURLResponse generate download URL response
type GenerateDownloadURLResponse struct {
	DownloadURL  string `json:"downloadUrl" example:"https://minio:9000/..."`
	ExpiresIn    int64  `json:"expiresIn" example:"3600"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty" example:"https://minio:9000/..."`
}

// ListFilesRequest list files request
type ListFilesRequest struct {
	FileType *string `form:"fileType" example:"image"`
	Page     int     `form:"page" binding:"required,min=1" example:"1"`
	PageSize int     `form:"pageSize" binding:"required,min=1,max=100" example:"20"`
}

// ListFilesResponse list files response
type ListFilesResponse struct {
	Files    []*FileInfoResponse `json:"files"`
	Total    int64               `json:"total" example:"100"`
	Page     int                 `json:"page" example:"1"`
	PageSize int                 `json:"pageSize" example:"20"`
}

// DeleteFileRequest delete file request (for internal use only)
type DeleteFileRequest struct {
	FileID string `json:"fileId" binding:"required" example:"file-123"`
}

// DeleteFileResponse delete file response
type DeleteFileResponse struct {
	Success bool `json:"success" example:"true"`
}
