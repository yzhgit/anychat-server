package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/file/model"
	"gorm.io/gorm"
)

// FileRepository 文件仓储接口
type FileRepository interface {
	// Create 创建文件记录
	Create(ctx context.Context, file *model.File) error

	// GetByFileID 根据file_id查询
	GetByFileID(ctx context.Context, fileID string) (*model.File, error)

	// GetByFileIDAndUserID 根据file_id和user_id查询（权限验证）
	GetByFileIDAndUserID(ctx context.Context, fileID, userID string) (*model.File, error)

	// BatchGetByFileIDs 批量查询文件
	BatchGetByFileIDs(ctx context.Context, fileIDs []string) ([]*model.File, error)

	// ListByUserID 列出用户文件（支持分页和类型过滤）
	ListByUserID(ctx context.Context, userID string, fileType *string, page, pageSize int) ([]*model.File, int64, error)

	// Update 更新文件信息
	Update(ctx context.Context, file *model.File) error

	// UpdateStatus 更新文件状态
	UpdateStatus(ctx context.Context, fileID string, status int16) error

	// Delete 软删除文件
	Delete(ctx context.Context, fileID string) error

	// DeleteExpired 清理过期文件
	DeleteExpired(ctx context.Context) error

	// WithTx 使用事务
	WithTx(tx *gorm.DB) FileRepository
}

// fileRepositoryImpl 文件仓储实现
type fileRepositoryImpl struct {
	db *gorm.DB
}

// NewFileRepository 创建文件仓储
func NewFileRepository(db *gorm.DB) FileRepository {
	return &fileRepositoryImpl{db: db}
}

// Create 创建文件记录
func (r *fileRepositoryImpl) Create(ctx context.Context, file *model.File) error {
	return r.db.WithContext(ctx).Create(file).Error
}

// GetByFileID 根据file_id查询
func (r *fileRepositoryImpl) GetByFileID(ctx context.Context, fileID string) (*model.File, error) {
	var file model.File
	err := r.db.WithContext(ctx).
		Where("file_id = ? AND status != ?", fileID, model.FileStatusDeleted).
		First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetByFileIDAndUserID 根据file_id和user_id查询（权限验证）
func (r *fileRepositoryImpl) GetByFileIDAndUserID(ctx context.Context, fileID, userID string) (*model.File, error) {
	var file model.File
	err := r.db.WithContext(ctx).
		Where("file_id = ? AND user_id = ? AND status != ?", fileID, userID, model.FileStatusDeleted).
		First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// BatchGetByFileIDs 批量查询文件
func (r *fileRepositoryImpl) BatchGetByFileIDs(ctx context.Context, fileIDs []string) ([]*model.File, error) {
	var files []*model.File
	err := r.db.WithContext(ctx).
		Where("file_id IN ? AND status != ?", fileIDs, model.FileStatusDeleted).
		Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, nil
}

// ListByUserID 列出用户文件（支持分页和类型过滤）
func (r *fileRepositoryImpl) ListByUserID(ctx context.Context, userID string, fileType *string, page, pageSize int) ([]*model.File, int64, error) {
	var files []*model.File
	var total int64

	query := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, model.FileStatusActive)

	// 按文件类型过滤
	if fileType != nil && *fileType != "" {
		query = query.Where("file_type = ?", *fileType)
	}

	// 统计总数
	if err := query.Model(&model.File{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&files).Error

	if err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

// Update 更新文件信息
func (r *fileRepositoryImpl) Update(ctx context.Context, file *model.File) error {
	return r.db.WithContext(ctx).Save(file).Error
}

// UpdateStatus 更新文件状态
func (r *fileRepositoryImpl) UpdateStatus(ctx context.Context, fileID string, status int16) error {
	return r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("file_id = ?", fileID).
		Update("status", status).Error
}

// Delete 软删除文件
func (r *fileRepositoryImpl) Delete(ctx context.Context, fileID string) error {
	return r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("file_id = ?", fileID).
		Update("status", model.FileStatusDeleted).Error
}

// DeleteExpired 清理过期文件
func (r *fileRepositoryImpl) DeleteExpired(ctx context.Context) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("expires_at IS NOT NULL AND expires_at < ? AND status != ?", now, model.FileStatusDeleted).
		Update("status", model.FileStatusDeleted).Error
}

// WithTx 使用事务
func (r *fileRepositoryImpl) WithTx(tx *gorm.DB) FileRepository {
	return &fileRepositoryImpl{db: tx}
}
