package grpc

import (
	"context"

	commonpb "github.com/anychat/server/api/proto/common"
	userpb "github.com/anychat/server/api/proto/user"
	"github.com/anychat/server/internal/user/dto"
	"github.com/anychat/server/internal/user/service"
	"github.com/anychat/server/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserServer user gRPC服务器
type UserServer struct {
	userpb.UnimplementedUserServiceServer
	userService service.UserService
}

// NewUserServer 创建user gRPC服务器
func NewUserServer(userService service.UserService) *UserServer {
	return &UserServer{
		userService: userService,
	}
}

// GetProfile 获取个人资料
func (s *UserServer) GetProfile(ctx context.Context, req *userpb.GetProfileRequest) (*userpb.UserProfileResponse, error) {
	resp, err := s.userService.GetProfile(ctx, req.UserId)
	if err != nil {
		return nil, convertError(err)
	}

	pbResp := &userpb.UserProfileResponse{
		UserId:     resp.UserID,
		Nickname:   resp.Nickname,
		Avatar:     resp.Avatar,
		Signature:  resp.Signature,
		Gender:     int32(resp.Gender),
		Region:     resp.Region,
		QrcodeUrl:  resp.QRCodeURL,
		CreatedAt:  timestamppb.New(resp.CreatedAt),
	}

	if resp.Birthday != nil {
		pbResp.Birthday = timestamppb.New(*resp.Birthday)
	}
	if resp.Phone != nil {
		pbResp.Phone = resp.Phone
	}
	if resp.Email != nil {
		pbResp.Email = resp.Email
	}

	return pbResp, nil
}

// UpdateProfile 更新个人资料
func (s *UserServer) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.UserProfileResponse, error) {
	dtoReq := &dto.UpdateProfileRequest{}

	if req.Nickname != nil {
		dtoReq.Nickname = req.Nickname
	}
	if req.Avatar != nil {
		dtoReq.Avatar = req.Avatar
	}
	if req.Signature != nil {
		dtoReq.Signature = req.Signature
	}
	if req.Gender != nil {
		gender := int(*req.Gender)
		dtoReq.Gender = &gender
	}
	if req.Birthday != nil {
		birthday := req.Birthday.AsTime()
		dtoReq.Birthday = &birthday
	}
	if req.Region != nil {
		dtoReq.Region = req.Region
	}

	resp, err := s.userService.UpdateProfile(ctx, req.UserId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	pbResp := &userpb.UserProfileResponse{
		UserId:     resp.UserID,
		Nickname:   resp.Nickname,
		Avatar:     resp.Avatar,
		Signature:  resp.Signature,
		Gender:     int32(resp.Gender),
		Region:     resp.Region,
		QrcodeUrl:  resp.QRCodeURL,
		CreatedAt:  timestamppb.New(resp.CreatedAt),
	}

	if resp.Birthday != nil {
		pbResp.Birthday = timestamppb.New(*resp.Birthday)
	}
	if resp.Phone != nil {
		pbResp.Phone = resp.Phone
	}
	if resp.Email != nil {
		pbResp.Email = resp.Email
	}

	return pbResp, nil
}

// GetUserInfo 获取用户信息
func (s *UserServer) GetUserInfo(ctx context.Context, req *userpb.GetUserInfoRequest) (*userpb.UserInfoResponse, error) {
	resp, err := s.userService.GetUserInfo(ctx, req.UserId, req.TargetUserId)
	if err != nil {
		return nil, convertError(err)
	}

	return &userpb.UserInfoResponse{
		UserId:    resp.UserID,
		Nickname:  resp.Nickname,
		Avatar:    resp.Avatar,
		Signature: resp.Signature,
		Gender:    int32(resp.Gender),
		Region:    resp.Region,
		IsFriend:  resp.IsFriend,
		IsBlocked: resp.IsBlocked,
	}, nil
}

// SearchUsers 搜索用户
func (s *UserServer) SearchUsers(ctx context.Context, req *userpb.SearchUsersRequest) (*userpb.SearchUsersResponse, error) {
	dtoReq := &dto.SearchUsersRequest{
		Keyword:  req.Keyword,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	resp, err := s.userService.SearchUsers(ctx, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	users := make([]*userpb.UserBriefInfo, 0, len(resp.Users))
	for _, u := range resp.Users {
		users = append(users, &userpb.UserBriefInfo{
			UserId:    u.UserID,
			Nickname:  u.Nickname,
			Avatar:    u.Avatar,
			Signature: u.Signature,
		})
	}

	return &userpb.SearchUsersResponse{
		Total: resp.Total,
		Users: users,
	}, nil
}

// GetSettings 获取用户设置
func (s *UserServer) GetSettings(ctx context.Context, req *userpb.GetSettingsRequest) (*userpb.UserSettingsResponse, error) {
	resp, err := s.userService.GetSettings(ctx, req.UserId)
	if err != nil {
		return nil, convertError(err)
	}

	return &userpb.UserSettingsResponse{
		UserId:                resp.UserID,
		NotificationEnabled:   resp.NotificationEnabled,
		SoundEnabled:          resp.SoundEnabled,
		VibrationEnabled:      resp.VibrationEnabled,
		MessagePreviewEnabled: resp.MessagePreviewEnabled,
		FriendVerifyRequired:  resp.FriendVerifyRequired,
		SearchByPhone:         resp.SearchByPhone,
		SearchById:            resp.SearchByID,
		Language:              resp.Language,
	}, nil
}

// UpdateSettings 更新用户设置
func (s *UserServer) UpdateSettings(ctx context.Context, req *userpb.UpdateSettingsRequest) (*userpb.UserSettingsResponse, error) {
	dtoReq := &dto.UpdateSettingsRequest{}

	if req.NotificationEnabled != nil {
		dtoReq.NotificationEnabled = req.NotificationEnabled
	}
	if req.SoundEnabled != nil {
		dtoReq.SoundEnabled = req.SoundEnabled
	}
	if req.VibrationEnabled != nil {
		dtoReq.VibrationEnabled = req.VibrationEnabled
	}
	if req.MessagePreviewEnabled != nil {
		dtoReq.MessagePreviewEnabled = req.MessagePreviewEnabled
	}
	if req.FriendVerifyRequired != nil {
		dtoReq.FriendVerifyRequired = req.FriendVerifyRequired
	}
	if req.SearchByPhone != nil {
		dtoReq.SearchByPhone = req.SearchByPhone
	}
	if req.SearchById != nil {
		dtoReq.SearchByID = req.SearchById
	}
	if req.Language != nil {
		dtoReq.Language = req.Language
	}

	resp, err := s.userService.UpdateSettings(ctx, req.UserId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &userpb.UserSettingsResponse{
		UserId:                resp.UserID,
		NotificationEnabled:   resp.NotificationEnabled,
		SoundEnabled:          resp.SoundEnabled,
		VibrationEnabled:      resp.VibrationEnabled,
		MessagePreviewEnabled: resp.MessagePreviewEnabled,
		FriendVerifyRequired:  resp.FriendVerifyRequired,
		SearchByPhone:         resp.SearchByPhone,
		SearchById:            resp.SearchByID,
		Language:              resp.Language,
	}, nil
}

// RefreshQRCode 刷新二维码
func (s *UserServer) RefreshQRCode(ctx context.Context, req *userpb.RefreshQRCodeRequest) (*userpb.QRCodeResponse, error) {
	resp, err := s.userService.RefreshQRCode(ctx, req.UserId)
	if err != nil {
		return nil, convertError(err)
	}

	return &userpb.QRCodeResponse{
		QrcodeUrl: resp.QRCodeURL,
		ExpiresAt: timestamppb.New(resp.ExpiresAt),
	}, nil
}

// GetUserByQRCode 通过二维码获取用户
func (s *UserServer) GetUserByQRCode(ctx context.Context, req *userpb.GetUserByQRCodeRequest) (*userpb.UserInfoResponse, error) {
	resp, err := s.userService.GetUserByQRCode(ctx, req.Qrcode)
	if err != nil {
		return nil, convertError(err)
	}

	return &userpb.UserInfoResponse{
		UserId:    resp.UserID,
		Nickname:  resp.Nickname,
		Avatar:    resp.Avatar,
		Signature: resp.Signature,
	}, nil
}

// UpdatePushToken 更新推送Token
func (s *UserServer) UpdatePushToken(ctx context.Context, req *userpb.UpdatePushTokenRequest) (*commonpb.Empty, error) {
	dtoReq := &dto.UpdatePushTokenRequest{
		DeviceID:  req.DeviceId,
		PushToken: req.PushToken,
		Platform:  req.Platform,
	}

	err := s.userService.UpdatePushToken(ctx, req.UserId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// InitUserData 初始化用户数据（供auth-service调用）
func (s *UserServer) InitUserData(ctx context.Context, req *userpb.InitUserDataRequest) (*commonpb.Empty, error) {
	err := s.userService.InitUserData(ctx, req.UserId, req.Nickname)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// convertError 将业务错误转换为gRPC错误
func convertError(err error) error {
	if bizErr, ok := err.(*errors.Business); ok {
		switch bizErr.Code {
		case errors.CodeParamError:
			return status.Error(codes.InvalidArgument, bizErr.Message)
		case errors.CodeUserProfileNotFound:
			return status.Error(codes.NotFound, bizErr.Message)
		case errors.CodeNicknameUsed:
			return status.Error(codes.AlreadyExists, bizErr.Message)
		case errors.CodeNicknameSensitive:
			return status.Error(codes.InvalidArgument, bizErr.Message)
		case errors.CodeQRCodeExpired, errors.CodeQRCodeInvalid:
			return status.Error(codes.InvalidArgument, bizErr.Message)
		default:
			return status.Error(codes.Internal, bizErr.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}
