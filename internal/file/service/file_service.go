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

// FileService file service interface
type FileService interface {
	// GenerateUploadToken generates upload token
	GenerateUploadToken(ctx context.Context, userID string, req *dto.GenerateUploadTokenRequest) (*dto.GenerateUploadTokenResponse, error)

	// CompleteUpload completes upload
	CompleteUpload(ctx context.Context, fileID, userID string) (*dto.FileInfoResponse, error)

	// GenerateDownloadURL generates download URL
	GenerateDownloadURL(ctx context.Context, fileID, userID string, expiresMinutes *int32) (*dto.GenerateDownloadURLResponse, error)

	// GetFileInfo gets file info
	GetFileInfo(ctx context.Context, fileID, userID string) (*dto.FileInfoResponse, error)

	// DeleteFile deletes file
	DeleteFile(ctx context.Context, fileID, userID string) error

	// ListUserFiles lists user files
	ListUserFiles(ctx context.Context, userID string, fileType *string, page, pageSize int) (*dto.ListFilesResponse, error)

	// BatchGetFileInfo batch gets file info
	BatchGetFileInfo(ctx context.Context, fileIDs []string, userID string) ([]*dto.FileInfoResponse, error)
}

// fileServiceImpl file service implementation
type fileServiceImpl struct {
	fileRepo    repository.FileRepository
	minioClient *minioclient.Client
	db          *gorm.DB
}

// NewFileService creates file service
func NewFileService(fileRepo repository.FileRepository, minioClient *minioclient.Client, db *gorm.DB) FileService {
	return &fileServiceImpl{
		fileRepo:    fileRepo,
		minioClient: minioClient,
		db:          db,
	}
}

// GenerateUploadToken generates upload token
func (s *fileServiceImpl) GenerateUploadToken(ctx context.Context, userID string, req *dto.GenerateUploadTokenRequest) (*dto.GenerateUploadTokenResponse, error) {
	// validate file size
	if err := s.validateFileSize(req.FileType, req.FileSize); err != nil {
		return nil, err
	}

	// generate unique file_id
	fileID := fmt.Sprintf("file-%s", uuid.New().String())

	// determine bucket
	bucketName := s.getBucketName(req.FileType)

	// generate storage path: {bucket}/{user_id}/{date}/{uuid}.{ext}
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	ext := s.getFileExtension(req.FileName)
	storagePath := fmt.Sprintf("%s/%s/%s.%s", userID, dateStr, uuid.New().String(), ext)

	// generate presigned upload URL (1 hour validity)
	uploadURL, err := s.minioClient.PresignedPutObject(ctx, bucketName, storagePath, time.Hour)
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeFileUploadFailed, "failed to generate upload URL")
	}

	// calculate expiration time
	var expiresAt *time.Time
	if req.ExpiresHours != nil && *req.ExpiresHours > 0 {
		expires := now.Add(time.Duration(*req.ExpiresHours) * time.Hour)
		expiresAt = &expires
	}

	// create file record (status is processing)
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
		ExpiresIn: 3600, // 1 hour
	}, nil
}

// CompleteUpload completes upload
func (s *fileServiceImpl) CompleteUpload(ctx context.Context, fileID, userID string) (*dto.FileInfoResponse, error) {
	// validate file record
	file, err := s.fileRepo.GetByFileIDAndUserID(ctx, fileID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeFileNotFound, "file not found")
		}
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to get file")
	}

	// validate file status
	if file.Status != model.FileStatusProcessing {
		return nil, errors.NewBusiness(errors.CodeInvalidFileID, "file already completed or deleted")
	}

	// validate file exists in MinIO
	_, err = s.minioClient.StatObject(ctx, file.BucketName, file.StoragePath)
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeFileUploadFailed, "file not found in storage")
	}

	// update status to active
	file.Status = model.FileStatusActive
	if err := s.fileRepo.Update(ctx, file); err != nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to update file status")
	}

	// return file info
	return s.toFileInfoResponse(file), nil
}

// GenerateDownloadURL generates download URL
func (s *fileServiceImpl) GenerateDownloadURL(ctx context.Context, fileID, userID string, expiresMinutes *int32) (*dto.GenerateDownloadURLResponse, error) {
	// validate permission
	file, err := s.fileRepo.GetByFileIDAndUserID(ctx, fileID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeFileNotFound, "file not found")
		}
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to get file")
	}

	// validate file status
	if file.Status != model.FileStatusActive {
		return nil, errors.NewBusiness(errors.CodeFileNotFound, "file is not active")
	}

	// check if file is expired
	if file.ExpiresAt != nil && file.ExpiresAt.Before(time.Now()) {
		return nil, errors.NewBusiness(errors.CodeFileExpired, "file has expired")
	}

	// generate download URL
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

	// if thumbnail exists, generate thumbnail URL
	if file.ThumbnailPath != "" {
		thumbnailURL, err := s.minioClient.PresignedGetObject(ctx, file.BucketName, file.ThumbnailPath, expires)
		if err == nil {
			resp.ThumbnailURL = thumbnailURL.String()
		}
	}

	return resp, nil
}

// GetFileInfo gets file info
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

// DeleteFile deletes file
func (s *fileServiceImpl) DeleteFile(ctx context.Context, fileID, userID string) error {
	// validate permission
	file, err := s.fileRepo.GetByFileIDAndUserID(ctx, fileID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeFileNotFound, "file not found")
		}
		return errors.NewBusiness(errors.CodeInternalError, "failed to get file")
	}

	// use transaction to delete
	return s.db.Transaction(func(tx *gorm.DB) error {
		fileRepo := s.fileRepo.WithTx(tx)

		// soft delete database record
		if err := fileRepo.Delete(ctx, fileID); err != nil {
			return errors.NewBusiness(errors.CodeInternalError, "failed to delete file record")
		}

		// delete MinIO object
		if err := s.minioClient.RemoveObject(ctx, file.BucketName, file.StoragePath); err != nil {
			// log error but don't rollback, since database record is already marked as deleted
			// TODO: can use async task to cleanup
		}

		// delete thumbnail
		if file.ThumbnailPath != "" {
			_ = s.minioClient.RemoveObject(ctx, file.BucketName, file.ThumbnailPath)
		}

		return nil
	})
}

// ListUserFiles lists user files
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

// BatchGetFileInfo batch gets file info
func (s *fileServiceImpl) BatchGetFileInfo(ctx context.Context, fileIDs []string, userID string) ([]*dto.FileInfoResponse, error) {
	if len(fileIDs) == 0 {
		return []*dto.FileInfoResponse{}, nil
	}

	files, err := s.fileRepo.BatchGetByFileIDs(ctx, fileIDs)
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "failed to batch get files")
	}

	// only return files that user has permission to (belongs to the user)
	fileInfos := make([]*dto.FileInfoResponse, 0, len(files))
	for _, file := range files {
		if file.UserID == userID {
			fileInfos = append(fileInfos, s.toFileInfoResponse(file))
		}
	}

	return fileInfos, nil
}

// validateFileSize validates file size
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

// getBucketName gets bucket name by file type
func (s *fileServiceImpl) getBucketName(fileType string) string {
	// currently all files are stored in chat-file bucket
	// in the future can separate by file type
	return model.BucketChatFile
}

// getFileExtension gets file extension
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

// toFileInfoResponse converts to DTO
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

	// convert metadata
	if file.Metadata.Width > 0 || file.Metadata.Height > 0 || file.Metadata.Duration > 0 || file.Metadata.Format != "" {
		metadataBytes, _ := json.Marshal(file.Metadata)
		metadata := make(map[string]string)
		_ = json.Unmarshal(metadataBytes, &metadata)
		resp.Metadata = metadata
	}

	return resp
}
