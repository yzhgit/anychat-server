package service

import (
	"context"
	"fmt"
	"time"

	"github.com/anychat/server/internal/user/dto"
	"github.com/anychat/server/internal/user/model"
	"github.com/anychat/server/internal/user/repository"
	"github.com/anychat/server/pkg/crypto"
	"github.com/anychat/server/pkg/errors"
	"github.com/anychat/server/pkg/validator"
	"gorm.io/gorm"
)

// UserService 用户服务接口
type UserService interface {
	// 用户资料
	GetProfile(ctx context.Context, userID string) (*dto.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userID string, req *dto.UpdateProfileRequest) (*dto.UserProfileResponse, error)
	GetUserInfo(ctx context.Context, userID, targetUserID string) (*dto.UserInfoResponse, error)
	SearchUsers(ctx context.Context, req *dto.SearchUsersRequest) (*dto.SearchUsersResponse, error)

	// 用户设置
	GetSettings(ctx context.Context, userID string) (*dto.UserSettingsResponse, error)
	UpdateSettings(ctx context.Context, userID string, req *dto.UpdateSettingsRequest) (*dto.UserSettingsResponse, error)

	// 二维码
	RefreshQRCode(ctx context.Context, userID string) (*dto.QRCodeResponse, error)
	GetUserByQRCode(ctx context.Context, code string) (*dto.UserBriefInfo, error)

	// 推送Token
	UpdatePushToken(ctx context.Context, userID string, req *dto.UpdatePushTokenRequest) error

	// 初始化用户数据
	InitUserData(ctx context.Context, userID, nickname string) error
}

// userServiceImpl 用户服务实现
type userServiceImpl struct {
	profileRepo   repository.UserProfileRepository
	settingsRepo  repository.UserSettingsRepository
	qrcodeRepo    repository.UserQRCodeRepository
	pushTokenRepo repository.UserPushTokenRepository
}

// NewUserService 创建用户服务
func NewUserService(
	profileRepo repository.UserProfileRepository,
	settingsRepo repository.UserSettingsRepository,
	qrcodeRepo repository.UserQRCodeRepository,
	pushTokenRepo repository.UserPushTokenRepository,
) UserService {
	return &userServiceImpl{
		profileRepo:   profileRepo,
		settingsRepo:  settingsRepo,
		qrcodeRepo:    qrcodeRepo,
		pushTokenRepo: pushTokenRepo,
	}
}

// InitUserData 初始化用户数据（注册时调用）
func (s *userServiceImpl) InitUserData(ctx context.Context, userID, nickname string) error {
	// 创建用户资料
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

	// 创建用户设置
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

	// 生成二维码
	_, err := s.RefreshQRCode(ctx, userID)
	return err
}

// GetProfile 获取个人资料
func (s *userServiceImpl) GetProfile(ctx context.Context, userID string) (*dto.UserProfileResponse, error) {
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeUserProfileNotFound, "")
		}
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

// UpdateProfile 更新个人资料
func (s *userServiceImpl) UpdateProfile(ctx context.Context, userID string, req *dto.UpdateProfileRequest) (*dto.UserProfileResponse, error) {
	// 获取当前资料
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 更新昵称
	if req.Nickname != nil {
		if !validator.ValidateNickname(*req.Nickname) {
			return nil, errors.NewBusiness(errors.CodeParamError, "昵称格式错误")
		}

		if validator.ContainsSensitiveWords(*req.Nickname) {
			return nil, errors.NewBusiness(errors.CodeNicknameSensitive, "")
		}

		// 检查昵称是否被占用
		exists, err := s.profileRepo.CheckNicknameExists(ctx, *req.Nickname, userID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.NewBusiness(errors.CodeNicknameUsed, "")
		}

		profile.Nickname = *req.Nickname
	}

	// 更新头像
	if req.Avatar != nil {
		profile.Avatar = *req.Avatar
	}

	// 更新签名
	if req.Signature != nil {
		profile.Signature = *req.Signature
	}

	// 更新性别
	if req.Gender != nil {
		if !validator.ValidateGender(*req.Gender) {
			return nil, errors.NewBusiness(errors.CodeParamError, "性别值无效")
		}
		profile.Gender = *req.Gender
	}

	// 更新生日
	if req.Birthday != nil {
		profile.Birthday = req.Birthday
	}

	// 更新地区
	if req.Region != nil {
		profile.Region = *req.Region
	}

	// 保存更新
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

// GetUserInfo 获取用户信息（查询其他用户）
func (s *userServiceImpl) GetUserInfo(ctx context.Context, userID, targetUserID string) (*dto.UserInfoResponse, error) {
	profile, err := s.profileRepo.GetByUserID(ctx, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeUserProfileNotFound, "")
		}
		return nil, err
	}

	// TODO: 查询好友关系和黑名单状态
	// 这需要调用Friend Service

	return &dto.UserInfoResponse{
		UserID:    profile.UserID,
		Nickname:  profile.Nickname,
		Avatar:    profile.Avatar,
		Signature: profile.Signature,
		Gender:    profile.Gender,
		Region:    profile.Region,
		IsFriend:  false, // TODO: 实际查询
		IsBlocked: false, // TODO: 实际查询
	}, nil
}

// SearchUsers 搜索用户
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

// GetSettings 获取个人设置
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

// UpdateSettings 更新个人设置
func (s *userServiceImpl) UpdateSettings(ctx context.Context, userID string, req *dto.UpdateSettingsRequest) (*dto.UserSettingsResponse, error) {
	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 更新设置
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

	// 保存更新
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

// RefreshQRCode 刷新二维码
func (s *userServiceImpl) RefreshQRCode(ctx context.Context, userID string) (*dto.QRCodeResponse, error) {
	// 生成二维码Token
	qrcodeToken, err := crypto.GenerateQRCodeToken(userID)
	if err != nil {
		return nil, err
	}

	// 生成二维码URL
	qrcodeURL := fmt.Sprintf("anychat://qrcode?token=%s", qrcodeToken)

	// 24小时后过期
	expiresAt := time.Now().Add(24 * time.Hour)

	// 保存二维码记录
	qrcode := &model.UserQRCode{
		UserID:      userID,
		QRCodeToken: qrcodeToken,
		QRCodeURL:   qrcodeURL,
		ExpiresAt:   expiresAt,
	}

	if err := s.qrcodeRepo.Create(ctx, qrcode); err != nil {
		return nil, err
	}

	// 更新用户资料中的二维码URL
	if err := s.profileRepo.UpdateQRCode(ctx, userID, qrcodeURL); err != nil {
		return nil, err
	}

	return &dto.QRCodeResponse{
		QRCodeURL: qrcodeURL,
		ExpiresAt: expiresAt,
	}, nil
}

// GetUserByQRCode 根据二维码获取用户信息
func (s *userServiceImpl) GetUserByQRCode(ctx context.Context, code string) (*dto.UserBriefInfo, error) {
	// 查找二维码
	qrcode, err := s.qrcodeRepo.GetByToken(ctx, code)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBusiness(errors.CodeQRCodeInvalid, "")
		}
		return nil, err
	}

	// 检查是否过期
	if qrcode.IsExpired() {
		return nil, errors.NewBusiness(errors.CodeQRCodeExpired, "")
	}

	// 获取用户资料
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

// UpdatePushToken 更新推送Token
func (s *userServiceImpl) UpdatePushToken(ctx context.Context, userID string, req *dto.UpdatePushTokenRequest) error {
	token := &model.UserPushToken{
		UserID:    userID,
		DeviceID:  req.DeviceID,
		PushToken: req.PushToken,
		Platform:  req.Platform,
	}

	return s.pushTokenRepo.CreateOrUpdate(ctx, token)
}
