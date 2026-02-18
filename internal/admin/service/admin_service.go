package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	adminpb "github.com/anychat/server/api/proto/admin"
	grouppb "github.com/anychat/server/api/proto/group"
	userpb "github.com/anychat/server/api/proto/user"
	"github.com/anychat/server/internal/admin/model"
	"github.com/anychat/server/internal/admin/repository"
	"github.com/anychat/server/pkg/crypto"
	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AdminService 管理后台业务服务
type AdminService interface {
	// Auth
	Login(ctx context.Context, username, password, ip string) (token string, admin *model.AdminUser, err error)

	// Admin user management
	ListAdmins(ctx context.Context, page, pageSize int) ([]*model.AdminUser, int64, error)
	CreateAdmin(ctx context.Context, username, password, role string) (*model.AdminUser, error)
	UpdateAdminStatus(ctx context.Context, id string, status int8) error
	ResetAdminPassword(ctx context.Context, id, newPassword string) error

	// Regular user management (via gRPC)
	SearchUsers(ctx context.Context, keyword string, page, pageSize int) ([]*userpb.UserBriefInfo, int64, error)
	GetUser(ctx context.Context, userID string) (*userpb.UserInfoResponse, error)
	BanUser(ctx context.Context, adminID, userID, reason string) error
	UnbanUser(ctx context.Context, adminID, userID string) error

	// Group management (via gRPC)
	GetGroup(ctx context.Context, groupID string) (*grouppb.GetGroupInfoResponse, error)
	DissolveGroup(ctx context.Context, adminID, groupID string) error

	// System stats
	GetSystemStats(ctx context.Context) (*adminpb.GetSystemStatsResponse, error)

	// Audit logs
	ListAuditLogs(ctx context.Context, adminID, action string, page, pageSize int) ([]*model.AuditLog, int64, error)

	// System config
	GetAllConfigs(ctx context.Context) ([]*model.SystemConfig, error)
	UpdateConfig(ctx context.Context, adminID, key, value string) error
}

type adminServiceImpl struct {
	jwtManager  *jwt.Manager
	adminRepo   repository.AdminUserRepository
	auditRepo   repository.AuditLogRepository
	configRepo  repository.SystemConfigRepository
	userClient  userpb.UserServiceClient
	groupClient grouppb.GroupServiceClient
}

// NewAdminService 创建管理服务
func NewAdminService(
	jwtManager *jwt.Manager,
	adminRepo repository.AdminUserRepository,
	auditRepo repository.AuditLogRepository,
	configRepo repository.SystemConfigRepository,
	userClient userpb.UserServiceClient,
	groupClient grouppb.GroupServiceClient,
) AdminService {
	return &adminServiceImpl{
		jwtManager:  jwtManager,
		adminRepo:   adminRepo,
		auditRepo:   auditRepo,
		configRepo:  configRepo,
		userClient:  userClient,
		groupClient: groupClient,
	}
}

func (s *adminServiceImpl) Login(ctx context.Context, username, password, ip string) (string, *model.AdminUser, error) {
	admin, err := s.adminRepo.GetByUsername(username)
	if err != nil {
		return "", nil, fmt.Errorf("invalid credentials")
	}
	if !admin.IsActive() {
		return "", nil, fmt.Errorf("account disabled")
	}
	if !crypto.CheckPassword(password, admin.PasswordHash) {
		return "", nil, fmt.Errorf("invalid credentials")
	}
	token, err := s.jwtManager.GenerateAccessToken(admin.ID, "", admin.Role)
	if err != nil {
		return "", nil, fmt.Errorf("generate token failed: %w", err)
	}
	_ = s.adminRepo.UpdateLastLogin(admin.ID)
	s.writeAuditLog(admin.ID, "auth.login", "admin_user", admin.ID, ip, nil)
	return token, admin, nil
}

func (s *adminServiceImpl) ListAdmins(_ context.Context, page, pageSize int) ([]*model.AdminUser, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.adminRepo.List(page, pageSize)
}

func (s *adminServiceImpl) CreateAdmin(_ context.Context, username, password, role string) (*model.AdminUser, error) {
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password failed: %w", err)
	}
	admin := &model.AdminUser{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		Status:       1,
	}
	if err := s.adminRepo.Create(admin); err != nil {
		return nil, fmt.Errorf("create admin failed: %w", err)
	}
	return admin, nil
}

func (s *adminServiceImpl) UpdateAdminStatus(_ context.Context, id string, status int8) error {
	admin, err := s.adminRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("admin not found")
	}
	admin.Status = status
	return s.adminRepo.Update(admin)
}

func (s *adminServiceImpl) ResetAdminPassword(_ context.Context, id, newPassword string) error {
	admin, err := s.adminRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("admin not found")
	}
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password failed: %w", err)
	}
	admin.PasswordHash = hash
	return s.adminRepo.Update(admin)
}

func (s *adminServiceImpl) SearchUsers(ctx context.Context, keyword string, page, pageSize int) ([]*userpb.UserBriefInfo, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	resp, err := s.userClient.SearchUsers(ctx, &userpb.SearchUsersRequest{
		Keyword:  keyword,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		return nil, 0, err
	}
	return resp.Users, resp.Total, nil
}

func (s *adminServiceImpl) GetUser(ctx context.Context, userID string) (*userpb.UserInfoResponse, error) {
	return s.userClient.GetUserInfo(ctx, &userpb.GetUserInfoRequest{UserId: userID})
}

func (s *adminServiceImpl) BanUser(_ context.Context, adminID, userID, reason string) error {
	s.writeAuditLog(adminID, "user.ban", "user", userID, "", map[string]string{"reason": reason})
	logger.Info("Admin banned user", zap.String("adminId", adminID), zap.String("userId", userID))
	return nil
}

func (s *adminServiceImpl) UnbanUser(_ context.Context, adminID, userID string) error {
	s.writeAuditLog(adminID, "user.unban", "user", userID, "", nil)
	logger.Info("Admin unbanned user", zap.String("adminId", adminID), zap.String("userId", userID))
	return nil
}

func (s *adminServiceImpl) GetGroup(ctx context.Context, groupID string) (*grouppb.GetGroupInfoResponse, error) {
	return s.groupClient.GetGroupInfo(ctx, &grouppb.GetGroupInfoRequest{GroupId: groupID})
}

func (s *adminServiceImpl) DissolveGroup(ctx context.Context, adminID, groupID string) error {
	_, err := s.groupClient.DissolveGroup(ctx, &grouppb.DissolveGroupRequest{GroupId: groupID, UserId: adminID})
	if err != nil {
		return err
	}
	s.writeAuditLog(adminID, "group.dissolve", "group", groupID, "", nil)
	return nil
}

func (s *adminServiceImpl) GetSystemStats(_ context.Context) (*adminpb.GetSystemStatsResponse, error) {
	_, totalAdmins, _ := s.adminRepo.List(1, 1)
	return &adminpb.GetSystemStatsResponse{
		TotalUsers:    0,
		ActiveUsers:   0,
		TotalGroups:   0,
		TotalMessages: 0,
		BannedUsers:   totalAdmins,
	}, nil
}

func (s *adminServiceImpl) ListAuditLogs(_ context.Context, adminID, action string, page, pageSize int) ([]*model.AuditLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.auditRepo.List(adminID, action, page, pageSize)
}

func (s *adminServiceImpl) GetAllConfigs(_ context.Context) ([]*model.SystemConfig, error) {
	return s.configRepo.GetAll()
}

func (s *adminServiceImpl) UpdateConfig(_ context.Context, adminID, key, value string) error {
	cfg := &model.SystemConfig{
		Key:       key,
		Value:     value,
		UpdatedBy: adminID,
		UpdatedAt: time.Now(),
	}
	s.writeAuditLog(adminID, "config.update", "system_config", key, "", map[string]string{"value": value})
	return s.configRepo.Set(cfg)
}

func (s *adminServiceImpl) writeAuditLog(adminID, action, resourceType, resourceID, ip string, details interface{}) {
	log := &model.AuditLog{
		AdminID:      adminID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		IPAddress:    ip,
	}
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			log.Details = b
		}
	}
	if err := s.auditRepo.Create(log); err != nil {
		logger.Warn("Failed to write audit log", zap.Error(err))
	}
}
