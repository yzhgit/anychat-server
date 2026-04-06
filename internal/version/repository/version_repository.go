package repository

import (
	"context"
	"time"

	"github.com/anychat/server/internal/version/model"
	"gorm.io/gorm"
)

type VersionRepository interface {
	Create(ctx context.Context, version *model.AppVersion) error
	GetByID(ctx context.Context, id int64) (*model.AppVersion, error)
	Update(ctx context.Context, version *model.AppVersion) error
	Delete(ctx context.Context, id int64) error
	GetLatestVersion(ctx context.Context, platform model.Platform, releaseType model.ReleaseType) (*model.AppVersion, error)
	ListVersions(ctx context.Context, platform model.Platform, releaseType model.ReleaseType, page, pageSize int) ([]*model.AppVersion, int64, error)
	Exists(ctx context.Context, platform model.Platform, version string, releaseType model.ReleaseType) (bool, error)
	IncrementStats(ctx context.Context, platform model.Platform, version string, date time.Time) error
}

type versionRepositoryImpl struct {
	db *gorm.DB
}

func NewVersionRepository(db *gorm.DB) VersionRepository {
	return &versionRepositoryImpl{db: db}
}

func (r *versionRepositoryImpl) Create(ctx context.Context, version *model.AppVersion) error {
	return r.db.WithContext(ctx).Create(version).Error
}

func (r *versionRepositoryImpl) GetByID(ctx context.Context, id int64) (*model.AppVersion, error) {
	var v model.AppVersion
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *versionRepositoryImpl) Update(ctx context.Context, version *model.AppVersion) error {
	return r.db.WithContext(ctx).Save(version).Error
}

func (r *versionRepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.AppVersion{}).Error
}

func (r *versionRepositoryImpl) GetLatestVersion(ctx context.Context, platform model.Platform, releaseType model.ReleaseType) (*model.AppVersion, error) {
	var v model.AppVersion
	query := r.db.WithContext(ctx).
		Where("platform = ? AND release_type = ? AND status = ?", platform, releaseType, model.VersionStatusPublished).
		Order("build_number DESC, id DESC").
		Limit(1)

	err := query.First(&v).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

func (r *versionRepositoryImpl) ListVersions(ctx context.Context, platform model.Platform, releaseType model.ReleaseType, page, pageSize int) ([]*model.AppVersion, int64, error) {
	var versions []*model.AppVersion
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AppVersion{})

	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if releaseType != "" {
		query = query.Where("release_type = ?", releaseType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Limit(pageSize).Offset(offset).Find(&versions).Error; err != nil {
		return nil, 0, err
	}

	return versions, total, nil
}

func (r *versionRepositoryImpl) Exists(ctx context.Context, platform model.Platform, version string, releaseType model.ReleaseType) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.AppVersion{}).
		Where("platform = ? AND version = ? AND release_type = ?", platform, version, releaseType).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *versionRepositoryImpl) IncrementStats(ctx context.Context, platform model.Platform, version string, date time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.ClientVersionStats{}).
		Where("platform = ? AND version = ? AND report_date = ?", platform, version, date).
		UpdateColumn("count", gorm.Expr("count + 1")).Error
}
