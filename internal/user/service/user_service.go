package service

import (
	"context"
	"fmt"
	"time"

	friendpb "github.com/anychat/server/api/proto/friend"
	authdto "github.com/anychat/server/internal/auth/dto"
	authmodel "github.com/anychat/server/internal/auth/model"
	authrepo "github.com/anychat/server/internal/auth/repository"
	authservice "github.com/anychat/server/internal/auth/service"
	"github.com/anychat/server/internal/user/dto"
	"github.com/anychat/server/internal/user/model"
	"github.com/anychat/server/internal/user/repository"
	"github.com/anychat/server/pkg/crypto"
	"github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/validator"
	"gorm.io/gorm"
)

// UserService user service interface
type UserService interface {
	// User profile
	GetProfile(ctx context.Context, userID string) (*dto.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userID string, req *dto.UpdateProfileRequest) (*dto.UserProfileResponse, error)
	GetUserInfo(ctx context.Context, userID, targetUserID string) (*dto.UserInfoResponse, error)
	SearchUsers(ctx context.Context, req *dto.SearchUsersRequest) (*dto.SearchUsersResponse, error)

	// User settings
	GetSettings(ctx context.Context, userID string) (*dto.UserSettingsResponse, error)
	UpdateSettings(ctx context.Context, userID string, req *dto.UpdateSettingsRequest) (*dto.UserSettingsResponse, error)

	// QR code
	RefreshQRCode(ctx context.Context, userID string) (*dto.QRCodeResponse, error)
	GetUserByQRCode(ctx context.Context, code string) (*dto.UserBriefInfo, error)

	// Push token
	UpdatePushToken(ctx context.Context, userID string, req *dto.UpdatePushTokenRequest) error

	// Account binding
	BindPhone(ctx context.Context, userID string, req *dto.BindPhoneRequest) (*dto.BindPhoneResponse, error)
	ChangePhone(ctx context.Context, userID string, req *dto.ChangePhoneRequest) (*dto.ChangePhoneResponse, error)
	BindEmail(ctx context.Context, userID string, req *dto.BindEmailRequest) (*dto.BindEmailResponse, error)
	ChangeEmail(ctx context.Context, userID string, req *dto.ChangeEmailRequest) (*dto.ChangeEmailResponse, error)

	// Initialize user data
	InitUserData(ctx context.Context, userID, nickname string) error
}

// userServiceImpl user service implementation
type userServiceImpl struct {
	profileRepo   repository.UserProfileRepository
	settingsRepo  repository.UserSettingsRepository
	qrcodeRepo    repository.UserQRCodeRepository
	pushTokenRepo repository.UserPushTokenRepository
	friendClient  friendpb.FriendServiceClient
	authUserRepo  authrepo.UserRepository
	sessionRepo   authrepo.UserSessionRepository
	verifySvc     authservice.VerificationService
}

// NewUserService creates user service
func NewUserService(
	profileRepo repository.UserProfileRepository,
	settingsRepo repository.UserSettingsRepository,
	qrcodeRepo repository.UserQRCodeRepository,
	pushTokenRepo repository.UserPushTokenRepository,
	friendClient friendpb.FriendServiceClient,
	authUserRepo authrepo.UserRepository,
	sessionRepo authrepo.UserSessionRepository,
	verifySvc authservice.VerificationService,
) UserService {
	return &userServiceImpl{
		profileRepo:   profileRepo,
		settingsRepo:  settingsRepo,
		qrcodeRepo:    qrcodeRepo,
		pushTokenRepo: pushTokenRepo,
		friendClient:  friendClient,
		authUserRepo:  authUserRepo,
		sessionRepo:   sessionRepo,
		verifySvc:     verifySvc,
	}
}

// InitUserData initializes user data (called during registration)
func (s *userServiceImpl) InitUserData(ctx context.Context, userID, nickname string) error {
	// Create user profile
	if nickname == "" {
		nickname = fmt.Sprintf("User_%s", userID[:8])
	}

	profile := &model.UserProfile{
		UserID:   userID,
		Nickname: nickname,
		Gender:   model.GenderUnknown,
	}

	if err := s.profileRepo.Create(ctx, profile); err != nil {
		return err
	}

	// Create user settings
	settings := &model.UserSettings{
		UserID:                userID,
		NotificationEnabled:   true,
		SoundEnabled:          true,
		VibrationEnabled:      true,
		MessagePreviewEnabled: true,
		FriendVerifyRequired:  true,
		SearchByPhone:         true,
		SearchByID:            true,
		Language:              "zh_CN",
	}

	if err := s.settingsRepo.Create(ctx, settings); err != nil {
		return err
	}

	// Generate QR code
	_, err := s.RefreshQRCode(ctx, userID)
	return err
}

// GetProfile retrieves personal profile
func (s *userServiceImpl) GetProfile(ctx context.Context, userID string) (*dto.UserProfileResponse, error) {
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeUserProfileNotFound, "")
		}
		return nil, err
	}

	resp := &dto.UserProfileResponse{
		UserID:    profile.UserID,
		Nickname:  profile.Nickname,
		Avatar:    profile.Avatar,
		Signature: profile.Signature,
		Gender:    profile.Gender,
		Birthday:  profile.Birthday,
		Region:    profile.Region,
		QRCodeURL: profile.QRCodeURL,
		CreatedAt: profile.CreatedAt,
	}

	if s.authUserRepo != nil {
		user, err := s.authUserRepo.GetByID(ctx, userID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.NewBusiness(errors.CodeUserNotFound, "")
			}
			return nil, err
		}
		resp.Phone = user.Phone
		resp.Email = user.Email
	}

	return resp, nil
}

// UpdateProfile updates personal profile
func (s *userServiceImpl) UpdateProfile(ctx context.Context, userID string, req *dto.UpdateProfileRequest) (*dto.UserProfileResponse, error) {
	// Get current profile
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update nickname
	if req.Nickname != nil {
		if !validator.ValidateNickname(*req.Nickname) {
			return nil, errors.NewBusiness(errors.CodeParamError, "invalid nickname format")
		}

		if validator.ContainsSensitiveWords(*req.Nickname) {
			return nil, errors.NewBusiness(errors.CodeNicknameSensitive, "")
		}

		// Check if nickname is already taken
		exists, err := s.profileRepo.CheckNicknameExists(ctx, *req.Nickname, userID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.NewBusiness(errors.CodeNicknameUsed, "")
		}

		profile.Nickname = *req.Nickname
	}

	// Update avatar
	if req.Avatar != nil {
		profile.Avatar = *req.Avatar
	}

	// Update signature
	if req.Signature != nil {
		profile.Signature = *req.Signature
	}

	// Update gender
	if req.Gender != nil {
		if !validator.ValidateGender(*req.Gender) {
			return nil, errors.NewBusiness(errors.CodeParamError, "invalid gender value")
		}
		profile.Gender = *req.Gender
	}

	// Update birthday
	if req.Birthday != nil {
		profile.Birthday = req.Birthday
	}

	// Update region
	if req.Region != nil {
		profile.Region = *req.Region
	}

	// Save update
	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return nil, err
	}

	return &dto.UserProfileResponse{
		UserID:    profile.UserID,
		Nickname:  profile.Nickname,
		Avatar:    profile.Avatar,
		Signature: profile.Signature,
		Gender:    profile.Gender,
		Birthday:  profile.Birthday,
		Region:    profile.Region,
		QRCodeURL: profile.QRCodeURL,
		CreatedAt: profile.CreatedAt,
	}, nil
}

// GetUserInfo retrieves user info (query other users)
func (s *userServiceImpl) GetUserInfo(ctx context.Context, userID, targetUserID string) (*dto.UserInfoResponse, error) {
	profile, err := s.profileRepo.GetByUserID(ctx, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeUserProfileNotFound, "")
		}
		return nil, err
	}

	isFriend := false
	isBlocked := false

	// Only query friend relationship and block status when requester exists and dependency is available
	if userID != "" && s.friendClient != nil {
		blockedResp, err := s.friendClient.IsBlocked(ctx, &friendpb.IsBlockedRequest{
			UserId:       userID,
			TargetUserId: targetUserID,
		})
		if err != nil {
			return nil, err
		}
		isBlocked = blockedResp.IsBlocked
		if isBlocked {
			return nil, errors.NewBusiness(errors.CodePermissionDenied, "you have been blocked by this user")
		}

		friendResp, err := s.friendClient.IsFriend(ctx, &friendpb.IsFriendRequest{
			UserId:   userID,
			FriendId: targetUserID,
		})
		if err != nil {
			return nil, err
		}
		isFriend = friendResp.IsFriend
	}

	return &dto.UserInfoResponse{
		UserID:    profile.UserID,
		Nickname:  profile.Nickname,
		Avatar:    profile.Avatar,
		Signature: profile.Signature,
		Gender:    profile.Gender,
		Region:    profile.Region,
		IsFriend:  isFriend,
		IsBlocked: isBlocked,
	}, nil
}

// SearchUsers searches for users
func (s *userServiceImpl) SearchUsers(ctx context.Context, req *dto.SearchUsersRequest) (*dto.SearchUsersResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize
	profiles, total, err := s.profileRepo.SearchByKeyword(ctx, req.Keyword, req.PageSize, offset)
	if err != nil {
		return nil, err
	}

	users := make([]*dto.UserBriefInfo, 0, len(profiles))
	for _, p := range profiles {
		users = append(users, &dto.UserBriefInfo{
			UserID:    p.UserID,
			Nickname:  p.Nickname,
			Avatar:    p.Avatar,
			Signature: p.Signature,
		})
	}

	return &dto.SearchUsersResponse{
		Total: total,
		Users: users,
	}, nil
}

// GetSettings retrieves personal settings
func (s *userServiceImpl) GetSettings(ctx context.Context, userID string) (*dto.UserSettingsResponse, error) {
	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeUserProfileNotFound, "")
		}
		return nil, err
	}

	return &dto.UserSettingsResponse{
		UserID:                settings.UserID,
		NotificationEnabled:   settings.NotificationEnabled,
		SoundEnabled:          settings.SoundEnabled,
		VibrationEnabled:      settings.VibrationEnabled,
		MessagePreviewEnabled: settings.MessagePreviewEnabled,
		FriendVerifyRequired:  settings.FriendVerifyRequired,
		SearchByPhone:         settings.SearchByPhone,
		SearchByID:            settings.SearchByID,
		Language:              settings.Language,
	}, nil
}

// UpdateSettings updates personal settings
func (s *userServiceImpl) UpdateSettings(ctx context.Context, userID string, req *dto.UpdateSettingsRequest) (*dto.UserSettingsResponse, error) {
	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update settings
	if req.NotificationEnabled != nil {
		settings.NotificationEnabled = *req.NotificationEnabled
	}
	if req.SoundEnabled != nil {
		settings.SoundEnabled = *req.SoundEnabled
	}
	if req.VibrationEnabled != nil {
		settings.VibrationEnabled = *req.VibrationEnabled
	}
	if req.MessagePreviewEnabled != nil {
		settings.MessagePreviewEnabled = *req.MessagePreviewEnabled
	}
	if req.FriendVerifyRequired != nil {
		settings.FriendVerifyRequired = *req.FriendVerifyRequired
	}
	if req.SearchByPhone != nil {
		settings.SearchByPhone = *req.SearchByPhone
	}
	if req.SearchByID != nil {
		settings.SearchByID = *req.SearchByID
	}
	if req.Language != nil {
		settings.Language = *req.Language
	}

	// Save update
	if err := s.settingsRepo.Update(ctx, settings); err != nil {
		return nil, err
	}

	return &dto.UserSettingsResponse{
		UserID:                settings.UserID,
		NotificationEnabled:   settings.NotificationEnabled,
		SoundEnabled:          settings.SoundEnabled,
		VibrationEnabled:      settings.VibrationEnabled,
		MessagePreviewEnabled: settings.MessagePreviewEnabled,
		FriendVerifyRequired:  settings.FriendVerifyRequired,
		SearchByPhone:         settings.SearchByPhone,
		SearchByID:            settings.SearchByID,
		Language:              settings.Language,
	}, nil
}

// RefreshQRCode refreshes QR code
func (s *userServiceImpl) RefreshQRCode(ctx context.Context, userID string) (*dto.QRCodeResponse, error) {
	// Generate QR code token
	qrcodeToken, err := crypto.GenerateQRCodeToken(userID)
	if err != nil {
		return nil, err
	}

	// Generate QR code URL
	qrcodeURL := fmt.Sprintf("anychat://qrcode?token=%s", qrcodeToken)

	// Expires after 24 hours
	expiresAt := time.Now().Add(24 * time.Hour)

	// Save QR code record
	qrcode := &model.UserQRCode{
		UserID:      userID,
		QRCodeToken: qrcodeToken,
		QRCodeURL:   qrcodeURL,
		ExpiresAt:   expiresAt,
	}

	if err := s.qrcodeRepo.Create(ctx, qrcode); err != nil {
		return nil, err
	}

	// Update QR code URL in user profile
	if err := s.profileRepo.UpdateQRCode(ctx, userID, qrcodeURL); err != nil {
		return nil, err
	}

	return &dto.QRCodeResponse{
		QRCodeURL: qrcodeURL,
		ExpiresAt: expiresAt,
	}, nil
}

// GetUserByQRCode retrieves user info by QR code
func (s *userServiceImpl) GetUserByQRCode(ctx context.Context, code string) (*dto.UserBriefInfo, error) {
	// Find QR code
	qrcode, err := s.qrcodeRepo.GetByToken(ctx, code)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeQRCodeInvalid, "")
		}
		return nil, err
	}

	// Check if expired
	if qrcode.IsExpired() {
		return nil, errors.NewBusiness(errors.CodeQRCodeExpired, "")
	}

	// Get user profile
	profile, err := s.profileRepo.GetByUserID(ctx, qrcode.UserID)
	if err != nil {
		return nil, err
	}

	return &dto.UserBriefInfo{
		UserID:    profile.UserID,
		Nickname:  profile.Nickname,
		Avatar:    profile.Avatar,
		Signature: profile.Signature,
	}, nil
}

// UpdatePushToken updates push token
func (s *userServiceImpl) UpdatePushToken(ctx context.Context, userID string, req *dto.UpdatePushTokenRequest) error {
	token := &model.UserPushToken{
		UserID:    userID,
		DeviceID:  req.DeviceID,
		PushToken: req.PushToken,
		Platform:  req.Platform,
	}

	return s.pushTokenRepo.CreateOrUpdate(ctx, token)
}

// BindPhone binds phone number
func (s *userServiceImpl) BindPhone(ctx context.Context, userID string, req *dto.BindPhoneRequest) (*dto.BindPhoneResponse, error) {
	if !validator.ValidatePhone(req.PhoneNumber) {
		return nil, errors.NewBusiness(errors.CodePhoneFormatInvalid, "")
	}

	user, err := s.requireAuthUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Phone != nil {
		if *user.Phone == req.PhoneNumber {
			return &dto.BindPhoneResponse{
				PhoneNumber: maskPhone(req.PhoneNumber),
				IsPrimary:   true,
			}, nil
		}
		return nil, errors.NewBusiness(errors.CodeParamError, "phone already bound, please use change phone API")
	}

	if err := s.ensurePhoneAvailable(ctx, req.PhoneNumber, userID); err != nil {
		return nil, err
	}
	if err := s.verifyCode(ctx, req.PhoneNumber, authmodel.TargetTypeSMS, authmodel.PurposeBindPhone, req.VerifyCode); err != nil {
		return nil, err
	}
	if err := s.authUserRepo.UpdatePhone(ctx, userID, &req.PhoneNumber); err != nil {
		return nil, err
	}

	return &dto.BindPhoneResponse{
		PhoneNumber: maskPhone(req.PhoneNumber),
		IsPrimary:   true,
	}, nil
}

// ChangePhone changes phone number
func (s *userServiceImpl) ChangePhone(ctx context.Context, userID string, req *dto.ChangePhoneRequest) (*dto.ChangePhoneResponse, error) {
	if !validator.ValidatePhone(req.NewPhoneNumber) {
		return nil, errors.NewBusiness(errors.CodePhoneFormatInvalid, "")
	}

	user, err := s.requireAuthUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Phone == nil || *user.Phone != req.OldPhoneNumber {
		return nil, errors.NewBusiness(errors.CodeOldPhoneNotMatch, "")
	}
	if req.NewPhoneNumber == req.OldPhoneNumber {
		return nil, errors.NewBusiness(errors.CodeParamError, "new phone number cannot be the same as old phone number")
	}

	if req.OldVerifyCode != nil && *req.OldVerifyCode != "" {
		if err := s.verifyCode(ctx, req.OldPhoneNumber, authmodel.TargetTypeSMS, authmodel.PurposeChangePhone, *req.OldVerifyCode); err != nil {
			return nil, err
		}
	}
	if err := s.ensurePhoneAvailable(ctx, req.NewPhoneNumber, userID); err != nil {
		return nil, err
	}
	if err := s.verifyCode(ctx, req.NewPhoneNumber, authmodel.TargetTypeSMS, authmodel.PurposeChangePhone, req.NewVerifyCode); err != nil {
		return nil, err
	}
	if err := s.authUserRepo.UpdatePhone(ctx, userID, &req.NewPhoneNumber); err != nil {
		return nil, err
	}
	if err := s.invalidateSessionsAfterContactChange(ctx, userID, req.DeviceID); err != nil {
		return nil, err
	}

	return &dto.ChangePhoneResponse{
		OldPhoneNumber: maskPhone(req.OldPhoneNumber),
		NewPhoneNumber: req.NewPhoneNumber,
	}, nil
}

// BindEmail binds email
func (s *userServiceImpl) BindEmail(ctx context.Context, userID string, req *dto.BindEmailRequest) (*dto.BindEmailResponse, error) {
	if !validator.ValidateEmail(req.Email) {
		return nil, errors.NewBusiness(errors.CodeEmailFormatInvalid, "")
	}

	user, err := s.requireAuthUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Email != nil {
		if *user.Email == req.Email {
			return &dto.BindEmailResponse{
				Email:     maskEmail(req.Email),
				IsPrimary: true,
			}, nil
		}
		return nil, errors.NewBusiness(errors.CodeParamError, "email already bound, please use change email API")
	}

	if err := s.ensureEmailAvailable(ctx, req.Email, userID); err != nil {
		return nil, err
	}
	if err := s.verifyCode(ctx, req.Email, authmodel.TargetTypeEmail, authmodel.PurposeBindEmail, req.VerifyCode); err != nil {
		return nil, err
	}
	if err := s.authUserRepo.UpdateEmail(ctx, userID, &req.Email); err != nil {
		return nil, err
	}

	return &dto.BindEmailResponse{
		Email:     maskEmail(req.Email),
		IsPrimary: true,
	}, nil
}

// ChangeEmail changes email
func (s *userServiceImpl) ChangeEmail(ctx context.Context, userID string, req *dto.ChangeEmailRequest) (*dto.ChangeEmailResponse, error) {
	if !validator.ValidateEmail(req.NewEmail) {
		return nil, errors.NewBusiness(errors.CodeEmailFormatInvalid, "")
	}

	user, err := s.requireAuthUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Email == nil || *user.Email != req.OldEmail {
		return nil, errors.NewBusiness(errors.CodeOldEmailNotMatch, "")
	}
	if req.NewEmail == req.OldEmail {
		return nil, errors.NewBusiness(errors.CodeParamError, "new email cannot be the same as old email")
	}

	if req.OldVerifyCode != nil && *req.OldVerifyCode != "" {
		if err := s.verifyCode(ctx, req.OldEmail, authmodel.TargetTypeEmail, authmodel.PurposeChangeEmail, *req.OldVerifyCode); err != nil {
			return nil, err
		}
	}
	if err := s.ensureEmailAvailable(ctx, req.NewEmail, userID); err != nil {
		return nil, err
	}
	if err := s.verifyCode(ctx, req.NewEmail, authmodel.TargetTypeEmail, authmodel.PurposeChangeEmail, req.NewVerifyCode); err != nil {
		return nil, err
	}
	if err := s.authUserRepo.UpdateEmail(ctx, userID, &req.NewEmail); err != nil {
		return nil, err
	}
	if err := s.invalidateSessionsAfterContactChange(ctx, userID, req.DeviceID); err != nil {
		return nil, err
	}

	return &dto.ChangeEmailResponse{
		OldEmail: maskEmail(req.OldEmail),
		NewEmail: req.NewEmail,
	}, nil
}

func (s *userServiceImpl) requireAuthUser(ctx context.Context, userID string) (*authmodel.User, error) {
	if s.authUserRepo == nil {
		return nil, errors.NewBusiness(errors.CodeInternalError, "account module not initialized")
	}

	user, err := s.authUserRepo.GetByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeUserNotFound, "")
		}
		return nil, err
	}
	return user, nil
}

func (s *userServiceImpl) ensurePhoneAvailable(ctx context.Context, phone, excludeUserID string) error {
	user, err := s.authUserRepo.GetByPhone(ctx, phone)
	if err == nil && user.ID != excludeUserID {
		return errors.NewBusiness(errors.CodePhoneAlreadyBound, "")
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func (s *userServiceImpl) ensureEmailAvailable(ctx context.Context, email, excludeUserID string) error {
	user, err := s.authUserRepo.GetByEmail(ctx, email)
	if err == nil && user.ID != excludeUserID {
		return errors.NewBusiness(errors.CodeEmailAlreadyBound, "")
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func (s *userServiceImpl) verifyCode(ctx context.Context, target, targetType, purpose, code string) error {
	if s.verifySvc == nil {
		return errors.NewBusiness(errors.CodeInternalError, "verification module not initialized")
	}

	_, err := s.verifySvc.VerifyCode(ctx, &authdto.VerifyCodeRequest{
		Target:     target,
		TargetType: targetType,
		Purpose:    purpose,
		Code:       code,
	})
	return err
}

func (s *userServiceImpl) invalidateSessionsAfterContactChange(ctx context.Context, userID, deviceID string) error {
	if s.sessionRepo == nil {
		return errors.NewBusiness(errors.CodeInternalError, "session module not initialized")
	}
	return s.sessionRepo.DeleteByUserIDExceptDeviceID(ctx, userID, deviceID)
}

func maskPhone(phone string) string {
	if len(phone) < 7 {
		return "***"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func maskEmail(email string) string {
	at := 0
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			at = i
			break
		}
	}
	if at <= 0 || at == len(email)-1 {
		return "***"
	}

	name := email[:at]
	domain := email[at+1:]
	if len(name) <= 2 {
		return "***@" + domain
	}
	return name[:2] + "***@" + domain
}
