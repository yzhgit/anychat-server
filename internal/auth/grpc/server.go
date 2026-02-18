package grpc

import (
	"context"

	"github.com/anychat/server/api/proto/auth"
	commonpb "github.com/anychat/server/api/proto/common"
	"github.com/anychat/server/internal/auth/dto"
	"github.com/anychat/server/internal/auth/service"
	"github.com/anychat/server/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthServer auth gRPC服务器
type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	authService service.AuthService
}

// NewAuthServer 创建auth gRPC服务器
func NewAuthServer(authService service.AuthService) *AuthServer {
	return &AuthServer{
		authService: authService,
	}
}

// Register 用户注册
func (s *AuthServer) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.RegisterRequest{
		Password:   req.Password,
		VerifyCode: req.VerifyCode,
		DeviceType: req.DeviceType,
		DeviceID:   req.DeviceId,
	}
	if req.PhoneNumber != nil {
		dtoReq.PhoneNumber = *req.PhoneNumber
	}
	if req.Email != nil {
		dtoReq.Email = *req.Email
	}
	if req.Nickname != nil {
		dtoReq.Nickname = *req.Nickname
	}

	// 调用service层
	resp, err := s.authService.Register(ctx, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	// DTO -> Proto 转换
	return &authpb.RegisterResponse{
		UserId:       resp.UserID,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
	}, nil
}

// Login 用户登录
func (s *AuthServer) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.LoginRequest{
		Account:    req.Account,
		Password:   req.Password,
		DeviceType: req.DeviceType,
		DeviceID:   req.DeviceId,
	}

	// 调用service层
	resp, err := s.authService.Login(ctx, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	// DTO -> Proto 转换
	pbResp := &authpb.LoginResponse{
		UserId:       resp.UserID,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
	}

	if resp.User != nil {
		pbResp.User = &commonpb.UserInfo{
			UserId:   resp.User.UserID,
			Nickname: resp.User.Nickname,
			Avatar:   resp.User.Avatar,
		}
		if resp.User.Phone != nil {
			pbResp.User.Phone = resp.User.Phone
		}
		if resp.User.Email != nil {
			pbResp.User.Email = resp.User.Email
		}
	}

	return pbResp, nil
}

// Logout 用户登出
func (s *AuthServer) Logout(ctx context.Context, req *authpb.LogoutRequest) (*commonpb.Empty, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.LogoutRequest{
		DeviceID: req.DeviceId,
	}

	// 调用service层
	err := s.authService.Logout(ctx, req.UserId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// RefreshToken 刷新访问令牌
func (s *AuthServer) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	}

	// 调用service层
	resp, err := s.authService.RefreshToken(ctx, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	// DTO -> Proto 转换
	return &authpb.RefreshTokenResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
	}, nil
}

// ChangePassword 修改密码
func (s *AuthServer) ChangePassword(ctx context.Context, req *authpb.ChangePasswordRequest) (*commonpb.Empty, error) {
	// Proto -> DTO 转换
	dtoReq := &dto.ChangePasswordRequest{
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}

	// 调用service层
	err := s.authService.ChangePassword(ctx, req.UserId, dtoReq)
	if err != nil {
		return nil, convertError(err)
	}

	return &commonpb.Empty{}, nil
}

// ValidateToken 验证Token（供gateway调用）
func (s *AuthServer) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	// 调用service层
	claims, err := s.authService.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return &authpb.ValidateTokenResponse{
			Valid: false,
		}, nil
	}

	// 返回验证结果
	return &authpb.ValidateTokenResponse{
		Valid:      true,
		UserId:     claims.UserID,
		DeviceId:   claims.DeviceID,
		DeviceType: claims.DeviceType,
	}, nil
}

// convertError 将业务错误转换为gRPC错误
func convertError(err error) error {
	if bizErr, ok := err.(*errors.Business); ok {
		switch bizErr.Code {
		case errors.CodeParamError:
			return status.Error(codes.InvalidArgument, bizErr.Message)
		case errors.CodeUserNotFound:
			return status.Error(codes.NotFound, bizErr.Message)
		case errors.CodeUnauthorized:
			return status.Error(codes.Unauthenticated, bizErr.Message)
		case errors.CodeUserExists:
			return status.Error(codes.AlreadyExists, bizErr.Message)
		case errors.CodePasswordError:
			return status.Error(codes.Unauthenticated, bizErr.Message)
		case errors.CodePasswordWeak:
			return status.Error(codes.InvalidArgument, bizErr.Message)
		case errors.CodeAccountDisabled:
			return status.Error(codes.PermissionDenied, bizErr.Message)
		case errors.CodeRefreshTokenInvalid, errors.CodeRefreshTokenExpired:
			return status.Error(codes.Unauthenticated, bizErr.Message)
		default:
			return status.Error(codes.Internal, bizErr.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}
