package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/anychat/server/internal/file/dto"
	"github.com/anychat/server/internal/file/model"
	"github.com/anychat/server/internal/file/repository"
	minioclient "github.com/anychat/server/pkg/minio"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/anychat/server/pkg/errors"
)

// FileService 文件服务接口
type FileService interface {
	// GenerateUploadToken 生成上传凭证
	GenerateUploadToken(ctx context.Context, userID string, req *dto.GenerateUploadTokenRequest) (*dto.GenerateUploadTokenResponse, error)

	// CompleteUpload 完成上传
	CompleteUpload(ctx context.Context, fileID, userID string) (*dto.FileInfoResponse, error)

	// GenerateDownloadURL 生成下载链接
	GenerateDownloadURL(ctx context.Context, fileID, userID string, expiresMinutes *int32) (*dto.GenerateDownloadURLResponse, error)

	// GetFileInfo 获取文件信息
	GetFileInfo(ctx context.Context, fileID, userID string) (*dto.FileInfoResponse, error)

	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, fileID, userID string) error

	// ListUserFiles 列出用户文件
	ListUserFiles(ctx context.Context, userID string, fileType *string, page, pageSize int) (*dto.ListFilesResponse, error)

	// BatchGetFileInfo 批量获取文件信息
	BatchGetFileInfo(ctx context.Context, fileIDs []string, userID string) ([]*dto.FileInfoResponse, error)
}

// fileServiceImpl 文件服务实现
type fileServiceImpl struct {
	fileRepo    repository.FileRepository
	minioClient *minioclient.Client
	db          *gorm.DB
}

// NewFileService 创建文件服务
func NewFileService(fileRepo repository.FileRepository, minioClient *minioclient.Client, db *gorm.DB) FileService {
	return &fileServiceImpl{
		fileRepo:    fileRepo,
		minioClient: minioClient,
		db:          db,
	}
}

// GenerateUploadToken 生成上传凭证
func (s *fileServiceImpl) GenerateUploadToken(ctx context.Context, userID string, req *dto.GenerateUploadTokenRequest) (*dto.GenerateUploadTokenResponse, error) {
	// 验证文件大小
	if err := s.validateFileSize(req.FileType, req.FileSize); err != nil {
		return nil, err
	}

	// 生成唯一file_id
	fileID := fmt.Sprintf("file-%s", uuid.New().String())

	// 确定bucket
	bucketName := s.getBucketName(req.FileType)

	// 生成存储路径：{bucket}/{user_id}/{date}/{uuid}.{ext}
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	ext := s.getFileExtension(req.FileName)
	storagePath := fmt.Sprintf("%s/%s/%s.%s", userID, dateStr, uuid.New().String(), ext)

	// 生成presigned上传URL（1小时有效）
	uploadURL, err := s.minioClient.PresignedPutObject(ctx, bucketName, storagePath, time.Hour)
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeFileUploadFailed, "failed to generate upload URL")
	}

	// 计算过期时间
	var expiresAt *time.Time
	if req.ExpiresHours != nil && *req.ExpiresHours > 0 {
		expires := now.Add(time.Duration(*req.ExpiresHours) * time.Hour)
		expiresAt = &expires
	}

	// 创建文件记录（状态为processing）
	file := &model.File{
		FileID:      fileID,
		UserID:      userID,
		FileName:    req.FileName,
		FileType:    req.FileType,
		FileSize:    req.FileSize,
		MimeType:    req.MimeType,
		StoragePath: storagePath,
		BucketName:  bucketName,
		Status:      model.FileStatusProcessing,
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to create file record")
	}

	return &dto.GenerateUploadTokenResponse{
		FileID:    fileID,
		UploadURL: uploadURL.String(),
		ExpiresIn: 3600, // 1小时
	}, nil
}

// CompleteUpload 完成上传
func (s *fileServiceImpl) CompleteUpload(ctx context.Context, fileID, userID string) (*dto.FileInfoResponse, error) {
	// 验证文件记录
	file, err := s.fileRepo.GetByFileIDAndUserID(ctx, fileID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeFileNotFound, "file not found")
		}
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to get file")
	}

	// 验证文件状态
	if file.Status != model.FileStatusProcessing {
		return nil, errors.NewBusiness(errors.CodeInvalidFileID, "file already completed or deleted")
	}

	// 验证MinIO中文件存在
	_, err = s.minioClient.StatObject(ctx, file.BucketName, file.StoragePath)
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeFileUploadFailed, "file not found in storage")
	}

	// 更新状态为active
	file.Status = model.FileStatusActive
	if err := s.fileRepo.Update(ctx, file); err != nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to update file status")
	}

	// 返回文件信息
	return s.toFileInfoResponse(file), nil
}

// GenerateDownloadURL 生成下载链接
func (s *fileServiceImpl) GenerateDownloadURL(ctx context.Context, fileID, userID string, expiresMinutes *int32) (*dto.GenerateDownloadURLResponse, error) {
	// 验证权限
	file, err := s.fileRepo.GetByFileIDAndUserID(ctx, fileID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeFileNotFound, "file not found")
		}
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to get file")
	}

	// 验证文件状态
	if file.Status != model.FileStatusActive {
		return nil, errors.NewBusiness(errors.CodeFileNotFound, "file is not active")
	}

	// 检查文件是否过期
	if file.ExpiresAt != nil && file.ExpiresAt.Before(time.Now()) {
		return nil, errors.NewBusiness(errors.CodeFileExpired, "file has expired")
	}

	// 生成下载URL
	expires := 60 * time.Minute
	if expiresMinutes != nil && *expiresMinutes > 0 {
		expires = time.Duration(*expiresMinutes) * time.Minute
	}

	downloadURL, err := s.minioClient.PresignedGetObject(ctx, file.BucketName, file.StoragePath, expires)
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to generate download URL")
	}

	resp := &dto.GenerateDownloadURLResponse{
		DownloadURL: downloadURL.String(),
		ExpiresIn:   int64(expires.Seconds()),
	}

	// 如果有缩略图，生成缩略图URL
	if file.ThumbnailPath != "" {
		thumbnailURL, err := s.minioClient.PresignedGetObject(ctx, file.BucketName, file.ThumbnailPath, expires)
		if err == nil {
			resp.ThumbnailURL = thumbnailURL.String()
		}
	}

	return resp, nil
}

// GetFileInfo 获取文件信息
func (s *fileServiceImpl) GetFileInfo(ctx context.Context, fileID, userID string) (*dto.FileInfoResponse, error) {
	file, err := s.fileRepo.GetByFileIDAndUserID(ctx, fileID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeFileNotFound, "file not found")
		}
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to get file")
	}

	return s.toFileInfoResponse(file), nil
}

// DeleteFile 删除文件
func (s *fileServiceImpl) DeleteFile(ctx context.Context, fileID, userID string) error {
	// 验证权限
	file, err := s.fileRepo.GetByFileIDAndUserID(ctx, fileID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeFileNotFound, "file not found")
		}
		return errors.NewBusiness(errors.CodeInternalError, "failed to get file")
	}

	// 使用事务删除
	return s.db.Transaction(func(tx *gorm.DB) error {
		fileRepo := s.fileRepo.WithTx(tx)

		// 软删除数据库记录
		if err := fileRepo.Delete(ctx, fileID); err != nil {
			return errors.NewBusiness(errors.CodeInternalError, "failed to delete file record")
		}

		// 删除MinIO对象
		if err := s.minioClient.RemoveObject(ctx, file.BucketName, file.StoragePath); err != nil {
			// 记录错误但不回滚，因为数据库记录已标记删除
			// TODO: 可以使用异步任务清理
		}

		// 删除缩略图
		if file.ThumbnailPath != "" {
			_ = s.minioClient.RemoveObject(ctx, file.BucketName, file.ThumbnailPath)
		}

		return nil
	})
}

// ListUserFiles 列出用户文件
func (s *fileServiceImpl) ListUserFiles(ctx context.Context, userID string, fileType *string, page, pageSize int) (*dto.ListFilesResponse, error) {
	files, total, err := s.fileRepo.ListByUserID(ctx, userID, fileType, page, pageSize)
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to list files")
	}

	fileInfos := make([]*dto.FileInfoResponse, 0, len(files))
	for _, file := range files {
		fileInfos = append(fileInfos, s.toFileInfoResponse(file))
	}

	return &dto.ListFilesResponse{
		Files:    fileInfos,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// BatchGetFileInfo 批量获取文件信息
func (s *fileServiceImpl) BatchGetFileInfo(ctx context.Context, fileIDs []string, userID string) ([]*dto.FileInfoResponse, error) {
	if len(fileIDs) == 0 {
		return []*dto.FileInfoResponse{}, nil
	}

	files, err := s.fileRepo.BatchGetByFileIDs(ctx, fileIDs)
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to batch get files")
	}

	// 只返回有权限的文件（属于该用户）
	fileInfos := make([]*dto.FileInfoResponse, 0, len(files))
	for _, file := range files {
		if file.UserID == userID {
			fileInfos = append(fileInfos, s.toFileInfoResponse(file))
		}
	}

	return fileInfos, nil
}

// validateFileSize 验证文件大小
func (s *fileServiceImpl) validateFileSize(fileType string, fileSize int64) error {
	var maxSize int64

	switch fileType {
	case model.FileTypeImage:
		maxSize = model.MaxImageSize
	case model.FileTypeVideo:
		maxSize = model.MaxVideoSize
	case model.FileTypeAudio:
		maxSize = model.MaxAudioSize
	case model.FileTypeFile:
		maxSize = model.MaxFileSize
	default:
		return errors.NewBusiness(errors.CodeFileTypeNotAllowed, "invalid file type")
	}

	if fileSize > maxSize {
		return errors.NewBusiness(errors.CodeFileSizeExceeded, fmt.Sprintf("file size exceeds limit: %d bytes", maxSize))
	}

	return nil
}

// getBucketName 根据文件类型获取bucket名称
func (s *fileServiceImpl) getBucketName(fileType string) string {
	// 当前所有文件都存储在chat-file bucket
	// 未来可以根据文件类型分bucket
	return model.BucketChatFile
}

// getFileExtension 获取文件扩展名
func (s *fileServiceImpl) getFileExtension(fileName string) string {
	ext := path.Ext(fileName)
	if ext != "" {
		ext = strings.TrimPrefix(ext, ".")
	}
	if ext == "" {
		ext = "bin"
	}
	return ext
}

// toFileInfoResponse 转换为DTO
func (s *fileServiceImpl) toFileInfoResponse(file *model.File) *dto.FileInfoResponse {
	resp := &dto.FileInfoResponse{
		FileID:        file.FileID,
		UserID:        file.UserID,
		FileName:      file.FileName,
		FileType:      file.FileType,
		FileSize:      file.FileSize,
		MimeType:      file.MimeType,
		StoragePath:   file.StoragePath,
		ThumbnailPath: file.ThumbnailPath,
		BucketName:    file.BucketName,
		Status:        int32(file.Status),
		CreatedAt:     file.CreatedAt.Unix(),
	}

	if file.ExpiresAt != nil {
		expiresAt := file.ExpiresAt.Unix()
		resp.ExpiresAt = &expiresAt
	}

	// 转换metadata
	if file.Metadata.Width > 0 || file.Metadata.Height > 0 || file.Metadata.Duration > 0 || file.Metadata.Format != "" {
		metadataBytes, _ := json.Marshal(file.Metadata)
		metadata := make(map[string]string)
		_ = json.Unmarshal(metadataBytes, &metadata)
		resp.Metadata = metadata
	}

	return resp
}
