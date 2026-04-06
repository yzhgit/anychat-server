package model

import (
	"time"
)

type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
	PlatformPC      Platform = "pc"
	PlatformWeb     Platform = "web"
	PlatformH5      Platform = "h5"
)

type ReleaseType string

const (
	ReleaseTypeStable ReleaseType = "stable"
	ReleaseTypeBeta   ReleaseType = "beta"
	ReleaseTypeAlpha  ReleaseType = "alpha"
)

type VersionStatus string

const (
	VersionStatusDraft     VersionStatus = "draft"
	VersionStatusPublished VersionStatus = "published"
	VersionStatusArchived  VersionStatus = "archived"
)

type AppVersion struct {
	ID             int64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Platform       Platform      `gorm:"column:platform;not null" json:"platform"`
	Version        string        `gorm:"column:version;not null" json:"version"`
	BuildNumber    int           `gorm:"column:build_number;default:0" json:"buildNumber"`
	VersionCode    int           `gorm:"column:version_code" json:"versionCode"`
	MinVersion     string        `gorm:"column:min_version" json:"minVersion"`
	MinBuildNumber int           `gorm:"column:min_build_number" json:"minBuildNumber"`
	ForceUpdate    bool          `gorm:"column:force_update;default:false" json:"forceUpdate"`
	ReleaseType    ReleaseType   `gorm:"column:release_type;default:stable" json:"releaseType"`
	Title          string        `gorm:"column:title" json:"title"`
	Content        string        `gorm:"column:content" json:"content"`
	DownloadURL    string        `gorm:"column:download_url" json:"downloadUrl"`
	FileSize       int64         `gorm:"column:file_size" json:"fileSize"`
	FileHash       string        `gorm:"column:file_hash" json:"fileHash"`
	PublishedAt    *time.Time    `gorm:"column:published_at" json:"publishedAt"`
	Status         VersionStatus `gorm:"column:status;default:draft" json:"status"`
	CreatedAt      time.Time     `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt      time.Time     `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt      *time.Time    `gorm:"column:deleted_at" json:"deletedAt"`
}

func (AppVersion) TableName() string {
	return "app_versions"
}

type ClientVersionStats struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Platform   Platform  `gorm:"column:platform;not null" json:"platform"`
	Version    string    `gorm:"column:version;not null" json:"version"`
	Count      int       `gorm:"column:count;default:0" json:"count"`
	ReportDate time.Time `gorm:"column:report_date;not null" json:"reportDate"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (ClientVersionStats) TableName() string {
	return "client_version_stats"
}
