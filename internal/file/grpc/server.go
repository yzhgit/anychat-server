package grpc

import (
	"context"
	"encoding/json"

	filepb "github.com/anychat/server/api/proto/file"
	"github.com/anychat/server/internal/file/dto"
	"github.com/anychat/server/internal/file/service"
	"github.com/anychat/server/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FileServer gRPC服务器
type FileServer struct {
	filepb.UnimplementedFileServiceServer
	fileService service.FileService
}

// NewFileServer 创建gRPC服务器
func NewFileServer(fileService service.FileService) *FileServer {
	return &FileServer{
		fileService: fileService,
	}
}

// GenerateUploadToken 生成上传凭证
func (s *FileServer) GenerateUploadToken(ctx context.Context, req *filepb.GenerateUploadTokenRequest) (*filepb.GenerateUploadTokenResponse, error) {
	dtoReq := &dto.GenerateUploadTokenRequest{
		FileName:     req.FileName,
		FileSize:     req.FileSize,
		MimeType:     req.MimeType,
		FileType:     req.FileType,
		ExpiresHours: req.ExpiresHours,
	}

	resp, err := s.fileService.GenerateUploadToken(ctx, req.UserId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &filepb.GenerateUploadTokenResponse{
		FileId:    resp.FileID,
		UploadUrl: resp.UploadURL,
		ExpiresIn: resp.ExpiresIn,
	}, nil
}

// CompleteUpload 完成上传
func (s *FileServer) CompleteUpload(ctx context.Context, req *filepb.CompleteUploadRequest) (*filepb.FileInfo, error) {
	resp, err := s.fileService.CompleteUpload(ctx, req.FileId, req.UserId)
	if err != nil {
		return nil, convertError(err)
	}

	return toProtoFileInfo(resp), nil
}

// GenerateDownloadURL 生成下载链接
func (s *FileServer) GenerateDownloadURL(ctx context.Context, req *filepb.GenerateDownloadURLRequest) (*filepb.GenerateDownloadURLResponse, error) {
	resp, err := s.fileService.GenerateDownloadURL(ctx, req.FileId, req.UserId, req.ExpiresMinutes)
	if err != nil {
		return nil, convertError(err)
	}

	pbResp := &filepb.GenerateDownloadURLResponse{
		DownloadUrl: resp.DownloadURL,
		ExpiresIn:   resp.ExpiresIn,
	}

	if resp.ThumbnailURL != "" {
		pbResp.ThumbnailUrl = &resp.ThumbnailURL
	}

	return pbResp, nil
}

// GetFileInfo 获取文件信息
func (s *FileServer) GetFileInfo(ctx context.Context, req *filepb.GetFileInfoRequest) (*filepb.FileInfo, error) {
	resp, err := s.fileService.GetFileInfo(ctx, req.FileId, req.UserId)
	if err != nil {
		return nil, convertError(err)
	}

	return toProtoFileInfo(resp), nil
}

// DeleteFile 删除文件
func (s *FileServer) DeleteFile(ctx context.Context, req *filepb.DeleteFileRequest) (*filepb.DeleteFileResponse, error) {
	err := s.fileService.DeleteFile(ctx, req.FileId, req.UserId)
	if err != nil {
		return nil, convertError(err)
	}

	return &filepb.DeleteFileResponse{
		Success: true,
	}, nil
}

// ListUserFiles 列出用户文件
func (s *FileServer) ListUserFiles(ctx context.Context, req *filepb.ListUserFilesRequest) (*filepb.ListUserFilesResponse, error) {
	resp, err := s.fileService.ListUserFiles(ctx, req.UserId, req.FileType, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, convertError(err)
	}

	files := make([]*filepb.FileInfo, 0, len(resp.Files))
	for _, file := range resp.Files {
		files = append(files, toProtoFileInfo(file))
	}

	return &filepb.ListUserFilesResponse{
		Files:    files,
		Total:    resp.Total,
		Page:     int32(resp.Page),
		PageSize: int32(resp.PageSize),
	}, nil
}

// BatchGetFileInfo 批量获取文件信息
func (s *FileServer) BatchGetFileInfo(ctx context.Context, req *filepb.BatchGetFileInfoRequest) (*filepb.BatchGetFileInfoResponse, error) {
	resp, err := s.fileService.BatchGetFileInfo(ctx, req.FileIds, req.UserId)
	if err != nil {
		return nil, convertError(err)
	}

	files := make([]*filepb.FileInfo, 0, len(resp))
	for _, file := range resp {
		files = append(files, toProtoFileInfo(file))
	}

	return &filepb.BatchGetFileInfoResponse{
		Files: files,
	}, nil
}

// toProtoFileInfo 转换为proto FileInfo
func toProtoFileInfo(file *dto.FileInfoResponse) *filepb.FileInfo {
	pbFile := &filepb.FileInfo{
		FileId:        file.FileID,
		UserId:        file.UserID,
		FileName:      file.FileName,
		FileType:      file.FileType,
		FileSize:      file.FileSize,
		MimeType:      file.MimeType,
		StoragePath:   file.StoragePath,
		ThumbnailPath: file.ThumbnailPath,
		BucketName:    file.BucketName,
		Status:        file.Status,
		CreatedAt:     file.CreatedAt,
	}

	if file.ExpiresAt != nil {
		pbFile.ExpiresAt = file.ExpiresAt
	}

	if file.Metadata != nil {
		metadataJSON, _ := json.Marshal(file.Metadata)
		metadata := string(metadataJSON)
		pbFile.Metadata = &metadata
	}

	if file.DownloadURL != "" {
		pbFile.DownloadUrl = &file.DownloadURL
	}

	if file.ThumbnailURL != "" {
		pbFile.ThumbnailUrl = &file.ThumbnailURL
	}

	return pbFile
}

// convertError 错误转换
func convertError(err error) error {
	if bizErr, ok := err.(*errors.Business); ok {
		switch bizErr.Code {
		case errors.CodeFileNotFound:
			return status.Error(codes.NotFound, bizErr.Message)
		case errors.CodeFileAccessDenied:
			return status.Error(codes.PermissionDenied, bizErr.Message)
		case errors.CodeFileSizeExceeded, errors.CodeFileTypeNotAllowed, errors.CodeInvalidFileID:
			return status.Error(codes.InvalidArgument, bizErr.Message)
		case errors.CodeFileUploadFailed:
			return status.Error(codes.Internal, bizErr.Message)
		case errors.CodeFileExpired:
			return status.Error(codes.FailedPrecondition, bizErr.Message)
		default:
			return status.Error(codes.Internal, bizErr.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}
