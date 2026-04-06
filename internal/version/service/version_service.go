package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	versionpb "github.com/anychat/server/api/proto/version"
	"github.com/anychat/server/internal/version/model"
	"github.com/anychat/server/internal/version/repository"
	pkgerrors "github.com/anychat/server/pkg/errors"
	pkgredis "github.com/anychat/server/pkg/redis"
)

type VersionService interface {
	CheckVersion(ctx context.Context, req *versionpb.CheckVersionRequest) (*versionpb.CheckVersionResponse, error)
	GetLatestVersion(ctx context.Context, req *versionpb.GetLatestVersionRequest) (*versionpb.GetLatestVersionResponse, error)
	ListVersions(ctx context.Context, req *versionpb.ListVersionsRequest) (*versionpb.ListVersionsResponse, error)
	CreateVersion(ctx context.Context, req *versionpb.CreateVersionRequest) (*versionpb.CreateVersionResponse, error)
	GetVersion(ctx context.Context, req *versionpb.GetVersionRequest) (*versionpb.GetVersionResponse, error)
	DeleteVersion(ctx context.Context, req *versionpb.DeleteVersionRequest) error
	ReportVersion(ctx context.Context, req *versionpb.ReportVersionRequest) error
}

type versionServiceImpl struct {
	repo  repository.VersionRepository
	cache *pkgredis.Client
}

func NewVersionService(repo repository.VersionRepository, cache *pkgredis.Client) VersionService {
	return &versionServiceImpl{
		repo:  repo,
		cache: cache,
	}
}

func (s *versionServiceImpl) CheckVersion(ctx context.Context, req *versionpb.CheckVersionRequest) (*versionpb.CheckVersionResponse, error) {
	platform := model.Platform(req.Platform)
	releaseType := model.ReleaseTypeStable

	latest, err := s.getLatestVersionFromCache(ctx, platform, releaseType)
	if err != nil {
		return nil, err
	}

	if latest == nil {
		return &versionpb.CheckVersionResponse{
			HasUpdate: false,
		}, nil
	}

	clientVersion := req.Version
	clientBuildNumber := int(req.BuildNumber)

	forceUpdate := s.shouldForceUpdate(clientVersion, clientBuildNumber, latest.MinVersion, latest.MinBuildNumber)
	hasUpdate := forceUpdate || s.hasNewVersion(clientVersion, clientBuildNumber, latest.Version, latest.BuildNumber)

	resp := &versionpb.CheckVersionResponse{
		HasUpdate:         hasUpdate,
		LatestVersion:     latest.Version,
		LatestBuildNumber: int32(latest.BuildNumber),
		ForceUpdate:       forceUpdate,
		MinVersion:        latest.MinVersion,
		MinBuildNumber:    int32(latest.MinBuildNumber),
	}

	if hasUpdate {
		resp.UpdateInfo = &versionpb.UpdateInfo{
			Title:       latest.Title,
			Content:     latest.Content,
			DownloadUrl: latest.DownloadURL,
			FileSize:    latest.FileSize,
			FileHash:    latest.FileHash,
		}
	}

	return resp, nil
}

func (s *versionServiceImpl) GetLatestVersion(ctx context.Context, req *versionpb.GetLatestVersionRequest) (*versionpb.GetLatestVersionResponse, error) {
	platform := model.Platform(req.Platform)
	releaseType := model.ReleaseType(req.ReleaseType)
	if releaseType == "" {
		releaseType = model.ReleaseTypeStable
	}

	latest, err := s.getLatestVersionFromCache(ctx, platform, releaseType)
	if err != nil {
		return nil, err
	}

	if latest == nil {
		return &versionpb.GetLatestVersionResponse{}, nil
	}

	return &versionpb.GetLatestVersionResponse{
		Version: s.toVersionInfo(latest),
	}, nil
}

func (s *versionServiceImpl) ListVersions(ctx context.Context, req *versionpb.ListVersionsRequest) (*versionpb.ListVersionsResponse, error) {
	platform := model.Platform(req.Platform)
	releaseType := model.ReleaseType(req.ReleaseType)
	page := int(req.Page)
	pageSize := int(req.PageSize)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	versions, total, err := s.repo.ListVersions(ctx, platform, releaseType, page, pageSize)
	if err != nil {
		return nil, err
	}

	infos := make([]*versionpb.VersionInfo, 0, len(versions))
	for _, v := range versions {
		infos = append(infos, s.toVersionInfo(v))
	}

	return &versionpb.ListVersionsResponse{
		Total:    int32(total),
		Versions: infos,
	}, nil
}

func (s *versionServiceImpl) CreateVersion(ctx context.Context, req *versionpb.CreateVersionRequest) (*versionpb.CreateVersionResponse, error) {
	platform := model.Platform(req.Platform)
	releaseType := model.ReleaseType(req.ReleaseType)
	if releaseType == "" {
		releaseType = model.ReleaseTypeStable
	}

	if !isValidVersion(req.Version) {
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeVersionFormatError, "")
	}

	exists, err := s.repo.Exists(ctx, platform, req.Version, releaseType)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, pkgerrors.NewBusiness(pkgerrors.CodeVersionAlreadyExists, "")
	}

	now := time.Now()
	version := &model.AppVersion{
		Platform:       platform,
		Version:        req.Version,
		BuildNumber:    int(req.BuildNumber),
		VersionCode:    int(req.VersionCode),
		MinVersion:     req.MinVersion,
		MinBuildNumber: int(req.MinBuildNumber),
		ForceUpdate:    req.ForceUpdate,
		ReleaseType:    releaseType,
		Title:          req.Title,
		Content:        req.Content,
		DownloadURL:    req.DownloadUrl,
		FileSize:       req.FileSize,
		FileHash:       req.FileHash,
		Status:         model.VersionStatusPublished,
		PublishedAt:    &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, version); err != nil {
		return nil, err
	}

	s.invalidateCache(ctx, platform, releaseType)

	return &versionpb.CreateVersionResponse{
		Id:       version.ID,
		Platform: string(platform),
		Version:  version.Version,
	}, nil
}

func (s *versionServiceImpl) GetVersion(ctx context.Context, req *versionpb.GetVersionRequest) (*versionpb.GetVersionResponse, error) {
	version, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &versionpb.GetVersionResponse{
		Version: s.toVersionInfo(version),
	}, nil
}

func (s *versionServiceImpl) DeleteVersion(ctx context.Context, req *versionpb.DeleteVersionRequest) error {
	version, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, req.Id); err != nil {
		return err
	}

	s.invalidateCache(ctx, version.Platform, version.ReleaseType)
	return nil
}

func (s *versionServiceImpl) ReportVersion(ctx context.Context, req *versionpb.ReportVersionRequest) error {
	platform := model.Platform(req.Platform)
	date := time.Now().Truncate(24 * time.Hour)

	return s.repo.IncrementStats(ctx, platform, req.Version, date)
}

func (s *versionServiceImpl) getLatestVersionFromCache(ctx context.Context, platform model.Platform, releaseType model.ReleaseType) (*model.AppVersion, error) {
	cacheKey := fmt.Sprintf("version:%s:latest:%s", platform, releaseType)

	if s.cache != nil {
		val, err := s.cache.Get(ctx, cacheKey)
		if err == nil {
			var version model.AppVersion
			if json.Unmarshal([]byte(val), &version) == nil {
				return &version, nil
			}
		}
	}

	version, err := s.repo.GetLatestVersion(ctx, platform, releaseType)
	if err != nil {
		return nil, err
	}

	if s.cache != nil && version != nil {
		data, _ := json.Marshal(version)
		s.cache.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	return version, nil
}

func (s *versionServiceImpl) invalidateCache(ctx context.Context, platform model.Platform, releaseType model.ReleaseType) {
	cacheKey := fmt.Sprintf("version:%s:latest:%s", platform, releaseType)
	if s.cache != nil {
		s.cache.Del(ctx, cacheKey)
	}
}

func (s *versionServiceImpl) toVersionInfo(v *model.AppVersion) *versionpb.VersionInfo {
	info := &versionpb.VersionInfo{
		Id:             v.ID,
		Platform:       string(v.Platform),
		Version:        v.Version,
		BuildNumber:    int32(v.BuildNumber),
		VersionCode:    int32(v.VersionCode),
		MinVersion:     v.MinVersion,
		MinBuildNumber: int32(v.MinBuildNumber),
		ForceUpdate:    v.ForceUpdate,
		ReleaseType:    string(v.ReleaseType),
		Title:          v.Title,
		Content:        v.Content,
		DownloadUrl:    v.DownloadURL,
		FileSize:       v.FileSize,
		FileHash:       v.FileHash,
	}
	if v.PublishedAt != nil {
		info.PublishedAt = v.PublishedAt.Format(time.RFC3339)
	}
	return info
}

func (s *versionServiceImpl) shouldForceUpdate(clientVersion string, clientBuildNumber int, minVersion string, minBuildNumber int) bool {
	if minVersion == "" && minBuildNumber == 0 {
		return false
	}

	if minVersion != "" && compareVersion(clientVersion, minVersion) < 0 {
		return true
	}

	if clientBuildNumber > 0 && minBuildNumber > 0 && clientBuildNumber < minBuildNumber {
		return true
	}

	return false
}

func (s *versionServiceImpl) hasNewVersion(clientVersion string, clientBuildNumber int, latestVersion string, latestBuildNumber int) bool {
	if compareVersion(clientVersion, latestVersion) < 0 {
		return true
	}

	if clientBuildNumber > 0 && latestBuildNumber > 0 && clientBuildNumber < latestBuildNumber {
		return true
	}

	return false
}

func compareVersion(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		p1 := 0
		p2 := 0
		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &p1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &p2)
		}

		if p1 > p2 {
			return 1
		}
		if p1 < p2 {
			return -1
		}
	}

	return 0
}

func isValidVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
