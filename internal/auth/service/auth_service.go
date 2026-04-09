package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anychat/server/internal/auth/client"
	"github.com/anychat/server/internal/auth/dto"
	"github.com/anychat/server/internal/auth/model"
	"github.com/anychat/server/internal/auth/repository"
	"github.com/anychat/server/pkg/crypto"
	"github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"github.com/anychat/server/pkg/validator"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AuthService authentication service interface
type AuthService interface {
	SendVerificationCode(ctx context.Context, req *dto.SendVerificationCodeRequest) (*dto.SendVerificationCodeResponse, error)
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error)
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)
	Logout(ctx context.Context, userID string, req *dto.LogoutRequest) error
	RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error)
	ChangePassword(ctx context.Context, userID string, req *dto.ChangePasswordRequest) error
	ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error
	ValidateToken(ctx context.Context, token string) (*jwt.Claims, error)
}

// authServiceImpl authentication service implementation
type authServiceImpl struct {
	userRepo        repository.UserRepository
	deviceRepo      repository.UserDeviceRepository
	sessionRepo     repository.UserSessionRepository
	jwtManager      *jwt.Manager
	userClient      *client.UserClient
	verifySvc       VerificationService
	notificationPub notification.Publisher
}

// NewAuthService creates authentication service
func NewAuthService(
	userRepo repository.UserRepository,
	deviceRepo repository.UserDeviceRepository,
	sessionRepo repository.UserSessionRepository,
	jwtManager *jwt.Manager,
	userClient *client.UserClient,
	verifySvc VerificationService,
	notificationPub notification.Publisher,
) AuthService {
	return &authServiceImpl{
		userRepo:        userRepo,
		deviceRepo:      deviceRepo,
		sessionRepo:     sessionRepo,
		jwtManager:      jwtManager,
		userClient:      userClient,
		verifySvc:       verifySvc,
		notificationPub: notificationPub,
	}
}

// SendVerificationCode sends verification code
func (s *authServiceImpl) SendVerificationCode(ctx context.Context, req *dto.SendVerificationCodeRequest) (*dto.SendVerificationCodeResponse, error) {
	if s.verifySvc == nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "verification service not initialized")
	}

	resp, err := s.verifySvc.SendCode(ctx, &dto.SendCodeRequest{
		Target:     req.Target,
		TargetType: req.TargetType,
		Purpose:    req.Purpose,
		DeviceID:   req.DeviceID,
	}, req.IPAddress)
	if err != nil {
		return nil, err
	}

	return &dto.SendVerificationCodeResponse{
		CodeID:    resp.CodeID,
		ExpiresIn: resp.ExpiresIn,
	}, nil
}

// Register user registration
func (s *authServiceImpl) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// validate parameters
	if req.PhoneNumber == "" && req.Email == "" {
		return nil, errors.NewBusiness(errors.CodeParamError, "phone or email required")
	}

	// validate phone number format
	if req.PhoneNumber != "" && !validator.ValidatePhone(req.PhoneNumber) {
		return nil, errors.NewBusiness(errors.CodeParamError, "invalid phone number format")
	}

	// validate email format
	if req.Email != "" && !validator.ValidateEmail(req.Email) {
		return nil, errors.NewBusiness(errors.CodeParamError, "invalid email format")
	}

	// validate password strength
	if !crypto.ValidatePasswordStrength(req.Password) {
		return nil, errors.NewBusiness(errors.CodePasswordWeak, "")
	}

	// validate device type
	if !validator.ValidateDeviceType(req.DeviceType) {
		return nil, errors.NewBusiness(errors.CodeParamError, "invalid device type")
	}

	// check if user already exists
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

	if s.verifySvc == nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "verification service not initialized")
	}

	target, targetType := s.resolveVerificationTarget(req)
	if _, err := s.verifySvc.VerifyCode(ctx, &dto.VerifyCodeRequest{
		Target:     target,
		TargetType: targetType,
		Purpose:    model.PurposeRegister,
		Code:       req.VerifyCode,
	}); err != nil {
		return nil, err
	}

	// generate password hash
	passwordHash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// create user
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

	// create device record
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

	// generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(userID, req.DeviceID, req.DeviceType)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(userID, req.DeviceID, req.DeviceType)
	if err != nil {
		return nil, err
	}

	// save session
	session := &model.UserSession{
		UserID:                userID,
		DeviceID:              req.DeviceID,
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  time.Now().Add(2 * time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	// call user-service to initialize user data
	if s.userClient != nil {
		if err := s.userClient.InitUserData(ctx, userID, req.Nickname); err != nil {
			// init failure should not block registration, just log error
			logger.Error("Failed to init user data", zap.Error(err), zap.String("userID", userID))
		}
	}

	return &dto.RegisterResponse{
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    7200, // 2 hours
	}, nil
}

// Login user login
func (s *authServiceImpl) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	// validate device type
	if !validator.ValidateDeviceType(req.DeviceType) {
		return nil, errors.NewBusiness(errors.CodeParamError, "invalid device type")
	}

	// find user
	user, err := s.userRepo.GetByAccount(ctx, req.Account)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeUserNotFound, "")
		}
		return nil, err
	}

	// verify password
	if !crypto.CheckPassword(req.Password, user.PasswordHash) {
		return nil, errors.NewBusiness(errors.CodePasswordError, "")
	}

	// check user status
	if !user.IsActive() {
		return nil, errors.NewBusiness(errors.CodeAccountDisabled, "")
	}

	// handle same type device login, force logout old devices
	if err := s.handleSameTypeDeviceKick(ctx, user.ID, req.DeviceID, req.DeviceType); err != nil {
		logger.Warn("Failed to handle same type device kick", zap.Error(err))
	}

	// update or create device record
	device, err := s.deviceRepo.GetByUserIDAndDeviceID(ctx, user.ID, req.DeviceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			device = &model.UserDevice{
				UserID:      user.ID,
				DeviceID:    req.DeviceID,
				DeviceType:  req.DeviceType,
				LastLoginIP: req.IpAddress,
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
		if err := s.deviceRepo.UpdateLastLogin(ctx, user.ID, req.DeviceID, req.IpAddress); err != nil {
			return nil, err
		}
	}

	// generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, req.DeviceID, req.DeviceType)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID, req.DeviceID, req.DeviceType)
	if err != nil {
		return nil, err
	}

	// update or create session
	session, err := s.sessionRepo.GetByUserIDAndDeviceID(ctx, user.ID, req.DeviceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			session = &model.UserSession{
				UserID:                user.ID,
				DeviceID:              req.DeviceID,
				AccessToken:           accessToken,
				RefreshToken:          refreshToken,
				AccessTokenExpiresAt:  time.Now().Add(2 * time.Hour),
				RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
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

// handleSameTypeDeviceKick handles same type device login, forces logout of old devices
func (s *authServiceImpl) handleSameTypeDeviceKick(ctx context.Context, userID, deviceID, deviceType string) error {
	devices, err := s.deviceRepo.GetByUserIDAndDeviceType(ctx, userID, deviceType)
	if err != nil {
		return err
	}

	for _, device := range devices {
		if device.DeviceID == deviceID {
			continue
		}

		if err := s.sessionRepo.DeleteByUserIDAndDeviceID(ctx, userID, device.DeviceID); err != nil {
			logger.Warn("Failed to delete old session", zap.Error(err), zap.String("deviceID", device.DeviceID))
		}

		if s.notificationPub != nil {
			notif := notification.NewNotification(
				notification.TypeAuthForceLogout,
				userID,
				notification.PriorityHigh,
			)
			notif.Payload = map[string]interface{}{
				"device_id":   device.DeviceID,
				"device_type": device.DeviceType,
				"reason":      "new_device_login",
			}
			if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
				logger.Warn("Failed to publish force logout notification", zap.Error(err))
			}
		}
	}

	return nil
}

// forceLogoutOtherDevices forces logout of other devices
func (s *authServiceImpl) forceLogoutOtherDevices(ctx context.Context, userID, excludeDeviceID, reason string) error {
	devices, err := s.deviceRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	for _, device := range devices {
		if device.DeviceID == excludeDeviceID {
			continue
		}

		if err := s.sessionRepo.DeleteByUserIDAndDeviceID(ctx, userID, device.DeviceID); err != nil {
			logger.Warn("Failed to delete session", zap.Error(err), zap.String("deviceID", device.DeviceID))
			continue
		}

		if s.notificationPub != nil {
			notif := notification.NewNotification(
				notification.TypeAuthForceLogout,
				userID,
				notification.PriorityHigh,
			)
			notif.Payload = map[string]interface{}{
				"device_id":   device.DeviceID,
				"device_type": device.DeviceType,
				"reason":      reason,
			}
			if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
				logger.Warn("Failed to publish force logout notification", zap.Error(err))
			}
		}
	}

	return nil
}

// Logout user logout
func (s *authServiceImpl) Logout(ctx context.Context, userID string, req *dto.LogoutRequest) error {
	// delete session
	return s.sessionRepo.DeleteByUserIDAndDeviceID(ctx, userID, req.DeviceID)
}

// RefreshToken refresh token
func (s *authServiceImpl) RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error) {
	// validate refresh token
	claims, err := s.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, errors.NewBusiness(errors.CodeRefreshTokenInvalid, "")
	}

	// find session
	session, err := s.sessionRepo.GetByRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeRefreshTokenInvalid, "")
		}
		return nil, err
	}

	// check expiration
	if session.IsRefreshTokenExpired() {
		return nil, errors.NewBusiness(errors.CodeRefreshTokenExpired, "")
	}

	// generate new tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(claims.UserID, claims.DeviceID, claims.DeviceType)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(claims.UserID, claims.DeviceID, claims.DeviceType)
	if err != nil {
		return nil, err
	}

	// update session
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

// ChangePassword change password
func (s *authServiceImpl) ChangePassword(ctx context.Context, userID string, req *dto.ChangePasswordRequest) error {
	// get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// verify old password
	if !crypto.CheckPassword(req.OldPassword, user.PasswordHash) {
		return errors.NewBusiness(errors.CodePasswordError, "incorrect old password")
	}

	// validate new password strength
	if !crypto.ValidatePasswordStrength(req.NewPassword) {
		return errors.NewBusiness(errors.CodePasswordWeak, "")
	}

	// generate new password hash
	passwordHash, err := crypto.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	// update password
	if err := s.userRepo.UpdatePassword(ctx, userID, passwordHash); err != nil {
		return err
	}

	// force logout other devices (excluding current device)
	return s.forceLogoutOtherDevices(ctx, userID, req.DeviceID, "password_changed")
}

// ResetPassword reset password (forgot password)
func (s *authServiceImpl) ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error {
	// determine account type
	targetType := model.TargetTypeSMS
	if isEmail(req.Account) {
		targetType = model.TargetTypeEmail
	}

	// verify verification code
	_, err := s.verifySvc.VerifyCode(ctx, &dto.VerifyCodeRequest{
		Target:     req.Account,
		TargetType: targetType,
		Code:       req.VerifyCode,
		Purpose:    model.PurposeResetPassword,
	})
	if err != nil {
		return err
	}

	// get user
	user, err := s.userRepo.GetByAccount(ctx, req.Account)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewBusiness(errors.CodeUserNotFound, "user not found")
		}
		return err
	}

	// validate new password strength
	if !crypto.ValidatePasswordStrength(req.NewPassword) {
		return errors.NewBusiness(errors.CodePasswordWeak, "password too weak")
	}

	// generate new password hash
	passwordHash, err := crypto.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	// update password
	err = s.userRepo.UpdatePassword(ctx, user.ID, passwordHash)
	if err != nil {
		return err
	}

	// invalidate all user sessions (force logout)
	return s.forceLogoutAllDevices(ctx, user.ID, "password_reset")
}

// forceLogoutAllDevices forces logout of all devices
func (s *authServiceImpl) forceLogoutAllDevices(ctx context.Context, userID, reason string) error {
	devices, err := s.deviceRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	for _, device := range devices {
		if err := s.sessionRepo.DeleteByUserIDAndDeviceID(ctx, userID, device.DeviceID); err != nil {
			logger.Warn("Failed to delete session", zap.Error(err), zap.String("deviceID", device.DeviceID))
			continue
		}

		if s.notificationPub != nil {
			notif := notification.NewNotification(
				notification.TypeAuthForceLogout,
				userID,
				notification.PriorityHigh,
			)
			notif.Payload = map[string]interface{}{
				"device_id":   device.DeviceID,
				"device_type": device.DeviceType,
				"reason":      reason,
			}
			if err := s.notificationPub.PublishToUser(userID, notif); err != nil {
				logger.Warn("Failed to publish force logout notification", zap.Error(err))
			}
		}
	}

	return nil
}

// ValidateToken validates token
func (s *authServiceImpl) ValidateToken(ctx context.Context, token string) (*jwt.Claims, error) {
	claims, err := s.jwtManager.ValidateAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	return claims, nil
}

func (s *authServiceImpl) resolveVerificationTarget(req *dto.RegisterRequest) (string, string) {
	if req.PhoneNumber != "" {
		return req.PhoneNumber, model.TargetTypeSMS
	}
	return req.Email, model.TargetTypeEmail
}

func isEmail(account string) bool {
	return strings.Contains(account, "@")
}
