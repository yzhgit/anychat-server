package handler

import (
	"net/http"

	versionpb "github.com/anychat/server/api/proto/version"
	"github.com/gin-gonic/gin"
)

type VersionHandler struct {
	clientManager interface {
		Version() versionpb.VersionServiceClient
	}
}

func NewVersionHandler(cm interface {
	Version() versionpb.VersionServiceClient
}) *VersionHandler {
	return &VersionHandler{clientManager: cm}
}

func (h *VersionHandler) CheckVersion(c *gin.Context) {
	platform := c.Query("platform")
	version := c.Query("version")
	buildNumber := c.DefaultQuery("buildNumber", "0")

	if platform == "" || version == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "platform and version are required",
		})
		return
	}

	// Convert buildNumber to int32
	var bn int32
	for _, ch := range buildNumber {
		if ch >= '0' && ch <= '9' {
			bn = bn*10 + int32(ch-'0')
		}
	}

	resp, err := h.clientManager.Version().CheckVersion(c.Request.Context(), &versionpb.CheckVersionRequest{
		Platform:    platform,
		Version:     version,
		BuildNumber: bn,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    2,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"hasUpdate":         resp.HasUpdate,
			"latestVersion":     resp.LatestVersion,
			"latestBuildNumber": resp.LatestBuildNumber,
			"forceUpdate":       resp.ForceUpdate,
			"minVersion":        resp.MinVersion,
			"minBuildNumber":    resp.MinBuildNumber,
			"updateInfo":        resp.UpdateInfo,
		},
	})
}

func (h *VersionHandler) GetLatestVersion(c *gin.Context) {
	platform := c.Query("platform")
	releaseType := c.DefaultQuery("releaseType", "stable")

	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "platform is required",
		})
		return
	}

	resp, err := h.clientManager.Version().GetLatestVersion(c.Request.Context(), &versionpb.GetLatestVersionRequest{
		Platform:    platform,
		ReleaseType: releaseType,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    2,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"version": resp.Version,
		},
	})
}

func (h *VersionHandler) ListVersions(c *gin.Context) {
	platform := c.Query("platform")
	releaseType := c.DefaultQuery("releaseType", "")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("pageSize", "20")

	var pageInt, pageSizeInt int32
	for _, ch := range page {
		if ch >= '0' && ch <= '9' {
			pageInt = pageInt*10 + int32(ch-'0')
		}
	}
	for _, ch := range pageSize {
		if ch >= '0' && ch <= '9' {
			pageSizeInt = pageSizeInt*10 + int32(ch-'0')
		}
	}

	resp, err := h.clientManager.Version().ListVersions(c.Request.Context(), &versionpb.ListVersionsRequest{
		Platform:    platform,
		Page:        pageInt,
		PageSize:    pageSizeInt,
		ReleaseType: releaseType,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    2,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"total":    resp.Total,
			"versions": resp.Versions,
		},
	})
}

func (h *VersionHandler) ReportVersion(c *gin.Context) {
	var req struct {
		Platform    string `json:"platform"`
		Version     string `json:"version"`
		BuildNumber int32  `json:"buildNumber"`
		DeviceID    string `json:"deviceId"`
		OsVersion   string `json:"osVersion"`
		SdkVersion  string `json:"sdkVersion"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "invalid request body",
		})
		return
	}

	if req.Platform == "" || req.Version == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "platform and version are required",
		})
		return
	}

	_, err := h.clientManager.Version().ReportVersion(c.Request.Context(), &versionpb.ReportVersionRequest{
		Platform:    req.Platform,
		Version:     req.Version,
		BuildNumber: req.BuildNumber,
		DeviceId:    req.DeviceID,
		OsVersion:   req.OsVersion,
		SdkVersion:  req.SdkVersion,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    2,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}
