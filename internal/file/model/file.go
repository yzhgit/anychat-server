package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// File 文件元数据模型
type File struct {
	ID            int64        `gorm:"column:id;primaryKey;autoIncrement"`
	FileID        string       `gorm:"column:file_id;not null;uniqueIndex"`
	UserID        string       `gorm:"column:user_id;not null"`
	FileName      string       `gorm:"column:file_name;not null"`
	FileType      string       `gorm:"column:file_type;not null"`
	FileSize      int64        `gorm:"column:file_size;not null"`
	MimeType      string       `gorm:"column:mime_type;not null"`
	StoragePath   string       `gorm:"column:storage_path;not null"`
	ThumbnailPath string       `gorm:"column:thumbnail_path"`
	BucketName    string       `gorm:"column:bucket_name;not null"`
	Status        int16        `gorm:"column:status;default:1"`
	CreatedAt     time.Time    `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	ExpiresAt     *time.Time   `gorm:"column:expires_at"`
	Metadata      FileMetadata `gorm:"column:metadata;type:jsonb"`
}

// FileMetadata 文件扩展元数据
type FileMetadata struct {
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Duration int    `json:"duration,omitempty"`
	Format   string `json:"format,omitempty"`
}

// Scan 实现sql.Scanner接口
func (m *FileMetadata) Scan(value interface{}) error {
	if value == nil {
		*m = FileMetadata{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, m)
}

// Value 实现driver.Valuer接口
func (m FileMetadata) Value() (driver.Value, error) {
	if m.Width == 0 && m.Height == 0 && m.Duration == 0 && m.Format == "" {
		return nil, nil
	}
	return json.Marshal(m)
}

// TableName 表名
func (File) TableName() string {
	return "files"
}

// FileUpload 文件上传追踪模型（用于分片上传）
type FileUpload struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UploadID       string    `gorm:"column:upload_id;not null;uniqueIndex"`
	UserID         string    `gorm:"column:user_id;not null"`
	FileName       string    `gorm:"column:file_name;not null"`
	FileSize       int64     `gorm:"column:file_size;not null"`
	ChunkSize      int64     `gorm:"column:chunk_size;not null"`
	UploadedChunks ChunkInfo `gorm:"column:uploaded_chunks;type:jsonb"`
	Status         string    `gorm:"column:status;default:pending"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
	CompletedAt    *time.Time
	ExpiresAt      time.Time `gorm:"column:expires_at;not null"`
}

// ChunkInfo 分片信息
type ChunkInfo struct {
	Chunks []int `json:"chunks"`
}

// Scan 实现sql.Scanner接口
func (c *ChunkInfo) Scan(value interface{}) error {
	if value == nil {
		*c = ChunkInfo{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, c)
}

// Value 实现driver.Valuer接口
func (c ChunkInfo) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// TableName 表名
func (FileUpload) TableName() string {
	return "file_uploads"
}

// 文件状态常量
const (
	FileStatusDeleted    = 0
	FileStatusActive     = 1
	FileStatusProcessing = 2
)

// 文件类型常量
const (
	FileTypeImage = "image"
	FileTypeVideo = "video"
	FileTypeAudio = "audio"
	FileTypeFile  = "file"
)

// Bucket名称常量
const (
	BucketAvatar      = "avatar"
	BucketGroupAvatar = "group-avatar"
	BucketChatFile    = "chat-file"
)

// 文件大小限制常量（字节）
const (
	MaxImageSize = 10 * 1024 * 1024   // 10MB
	MaxVideoSize = 100 * 1024 * 1024  // 100MB
	MaxAudioSize = 20 * 1024 * 1024   // 20MB
	MaxFileSize  = 50 * 1024 * 1024   // 50MB
)

// 上传状态常量
const (
	UploadStatusPending   = "pending"
	UploadStatusUploading = "uploading"
	UploadStatusCompleted = "completed"
	UploadStatusFailed    = "failed"
)
