package service

import (
	"context"
	"fmt"
	"time"

	"github.com/anychat/server/internal/auth/client"
	"github.com/anychat/server/internal/auth/dto"
	"github.com/anychat/server/internal/auth/model"
	"github.com/anychat/server/internal/auth/repository"
	"github.com/anychat/server/pkg/crypto"
	"github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/validator"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AuthService 认证服务接口
type AuthService interface {
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error)
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)
	Logout(ctx context.Context, userID string, req *dto.LogoutRequest) error
	RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error)
	ChangePassword(ctx context.Context, userID string, req *dto.ChangePasswordRequest) error
	ValidateToken(ctx context.Context, token string) (*jwt.Claims, error)
}

// authServiceImpl 认证服务实现
type authServiceImpl struct {
	userRepo    repository.UserRepository
	deviceRepo  repository.UserDeviceRepository
	sessionRepo repository.UserSessionRepository
	jwtManager  *jwt.Manager
	userClient  *client.UserClient
}

// NewAuthService 创建认证服务
func NewAuthService(
	userRepo repository.UserRepository,
	deviceRepo repository.UserDeviceRepository,
	sessionRepo repository.UserSessionRepository,
	jwtManager *jwt.Manager,
	userClient *client.UserClient,
) AuthService {
	return &authServiceImpl{
		userRepo:    userRepo,
		deviceRepo:  deviceRepo,
		sessionRepo: sessionRepo,
		jwtManager:  jwtManager,
		userClient:  userClient,
	}
}

// Register 用户注册
func (s *authServiceImpl) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// 验证参数
	if req.PhoneNumber == "" && req.Email == "" {
		return nil, errors.NewBusiness(errors.CodeParamError, "手机号或邮箱至少填写一个")
	}

	// 验证手机号格式
	if req.PhoneNumber != "" && !validator.ValidatePhone(req.PhoneNumber) {
		return nil, errors.NewBusiness(errors.CodeParamError, "手机号格式错误")
	}

	// 验证邮箱格式
	if req.Email != "" && !validator.ValidateEmail(req.Email) {
		return nil, errors.NewBusiness(errors.CodeParamError, "邮箱格式错误")
	}

	// 验证密码强度
	if !crypto.ValidatePasswordStrength(req.Password) {
		return nil, errors.NewBusiness(errors.CodePasswordWeak, "")
	}

	// 验证设备类型
	if !validator.ValidateDeviceType(req.DeviceType) {
		return nil, errors.NewBusiness(errors.CodeParamError, "设备类型无效")
	}

	// TODO: 验证验证码（需要Redis集成）
	// 这里暂时跳过验证码校验

	// 检查用户是否已存在
	if req.PhoneNumber != "" {
		if _, err := s.userRepo.GetByPhone(ctx, req.PhoneNumber); err == nil {
			return nil, errors.NewBusiness(errors.CodeUserExists, "")
		} else if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	if req.Email != "" {
		if _, err := s.userRepo.GetByEmail(ctx, req.Email); err == nil {
			return nil, errors.NewBusiness(errors.CodeUserExists, "")
		} else if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	// 生成密码哈希
	passwordHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 创建用户
	userID := uuid.New().String()
	user := &model.User{
		ID:           userID,
		PasswordHash: passwordHash,
		Status:       model.UserStatusNormal,
	}

	if req.PhoneNumber != "" {
		user.Phone = &req.PhoneNumber
	}
	if req.Email != "" {
		user.Email = &req.Email
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 创建设备记录
	device := &model.UserDevice{
		UserID:     userID,
		DeviceID:   req.DeviceID,
		DeviceType: req.DeviceType,
	}
	now := time.Now()
	device.LastLoginAt = &now
	if err := s.deviceRepo.Create(ctx, device); err != nil {
		return nil, err
	}

	// 生成Token
	accessToken, err := s.jwtManager.GenerateAccessToken(userID, req.DeviceID, req.DeviceType)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(userID, req.DeviceID, req.DeviceType)
	if err != nil {
		return nil, err
	}

	// 保存会话
	session := &model.UserSession{
		UserID:                 userID,
		DeviceID:               req.DeviceID,
		AccessToken:            accessToken,
		RefreshToken:           refreshToken,
		AccessTokenExpiresAt:   time.Now().Add(2 * time.Hour),
		RefreshTokenExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	// 调用user-service初始化用户数据
	if s.userClient != nil {
		if err := s.userClient.InitUserData(ctx, userID, req.Nickname); err != nil {
			// 初始化失败不影响注册，记录错误即可
			logger.Error("Failed to init user data", zap.Error(err), zap.String("userID", userID))
		}
	}

	return &dto.RegisterResponse{
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    7200, // 2小时
	}, nil
}

// Login 用户登录
func (s *authServiceImpl) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	// 验证设备类型
	if !validator.ValidateDeviceType(req.DeviceType) {
		return nil, errors.NewBusiness(errors.CodeParamError, "设备类型无效")
	}

	// 查找用户
	user, err := s.userRepo.GetByAccount(ctx, req.Account)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeUserNotFound, "")
		}
		return nil, err
	}

	// 验证密码
	if !crypto.CheckPassword(req.Password, user.PasswordHash) {
		return nil, errors.NewBusiness(errors.CodePasswordError, "")
	}

	// 检查用户状态
	if !user.IsActive() {
		return nil, errors.NewBusiness(errors.CodeAccountDisabled, "")
	}

	// 更新或创建设备记录
	device, err := s.deviceRepo.GetByUserIDAndDeviceID(ctx, user.ID, req.DeviceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			device = &model.UserDevice{
				UserID:     user.ID,
				DeviceID:   req.DeviceID,
				DeviceType: req.DeviceType,
			}
			now := time.Now()
			device.LastLoginAt = &now
			if err := s.deviceRepo.Create(ctx, device); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		if err := s.deviceRepo.UpdateLastLogin(ctx, user.ID, req.DeviceID, ""); err != nil {
			return nil, err
		}
	}

	// 生成Token
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, req.DeviceID, req.DeviceType)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID, req.DeviceID, req.DeviceType)
	if err != nil {
		return nil, err
	}

	// 更新或创建会话
	session, err := s.sessionRepo.GetByUserIDAndDeviceID(ctx, user.ID, req.DeviceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			session = &model.UserSession{
				UserID:                 user.ID,
				DeviceID:               req.DeviceID,
				AccessToken:            accessToken,
				RefreshToken:           refreshToken,
				AccessTokenExpiresAt:   time.Now().Add(2 * time.Hour),
				RefreshTokenExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
			}
			if err := s.sessionRepo.Create(ctx, session); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		session.AccessToken = accessToken
		session.RefreshToken = refreshToken
		session.AccessTokenExpiresAt = time.Now().Add(2 * time.Hour)
		session.RefreshTokenExpiresAt = time.Now().Add(7 * 24 * time.Hour)
		if err := s.sessionRepo.Update(ctx, session); err != nil {
			return nil, err
		}
	}

	return &dto.LoginResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    7200,
		User: &dto.UserInfo{
			UserID: user.ID,
			Phone:  user.Phone,
			Email:  user.Email,
		},
	}, nil
}

// Logout 用户登出
func (s *authServiceImpl) Logout(ctx context.Context, userID string, req *dto.LogoutRequest) error {
	// 删除会话
	return s.sessionRepo.DeleteByUserIDAndDeviceID(ctx, userID, req.DeviceID)
}

// RefreshToken 刷新Token
func (s *authServiceImpl) RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error) {
	// 验证RefreshToken
	claims, err := s.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeRefreshTokenInvalid, "")
	}

	// 查找会话
	session, err := s.sessionRepo.GetByRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeRefreshTokenInvalid, "")
		}
		return nil, err
	}

	// 检查过期时间
	if session.IsRefreshTokenExpired() {
		return nil, errors.NewBusiness(errors.CodeRefreshTokenExpired, "")
	}

	// 生成新Token
	accessToken, err := s.jwtManager.GenerateAccessToken(claims.UserID, claims.DeviceID, claims.DeviceType)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(claims.UserID, claims.DeviceID, claims.DeviceType)
	if err != nil {
		return nil, err
	}

	// 更新会话
	session.AccessToken = accessToken
	session.RefreshToken = refreshToken
	session.AccessTokenExpiresAt = time.Now().Add(2 * time.Hour)
	session.RefreshTokenExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	return &dto.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    7200,
	}, nil
}

// ChangePassword 修改密码
func (s *authServiceImpl) ChangePassword(ctx context.Context, userID string, req *dto.ChangePasswordRequest) error {
	// 获取用户
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 验证旧密码
	if !crypto.CheckPassword(req.OldPassword, user.PasswordHash) {
		return errors.NewBusiness(errors.CodePasswordError, "旧密码错误")
	}

	// 验证新密码强度
	if !crypto.ValidatePasswordStrength(req.NewPassword) {
		return errors.NewBusiness(errors.CodePasswordWeak, "")
	}

	// 生成新密码哈希
	passwordHash, err := crypto.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	// 更新密码
	return s.userRepo.UpdatePassword(ctx, userID, passwordHash)
}

// ValidateToken 验证Token
func (s *authServiceImpl) ValidateToken(ctx context.Context, token string) (*jwt.Claims, error) {
	claims, err := s.jwtManager.ValidateAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	return claims, nil
}
