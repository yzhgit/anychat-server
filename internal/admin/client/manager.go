package client

import (
	"fmt"

	filepb "github.com/anychat/server/api/proto/file"
	grouppb "github.com/anychat/server/api/proto/group"
	userpb "github.com/anychat/server/api/proto/user"
	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Manager 下游gRPC客户端管理器
type Manager struct {
	userConn    *grpc.ClientConn
	groupConn   *grpc.ClientConn
	fileConn    *grpc.ClientConn
	UserClient  userpb.UserServiceClient
	GroupClient grouppb.GroupServiceClient
	FileClient  filepb.FileServiceClient
}

// NewManager 创建客户端管理器
func NewManager(userAddr, groupAddr, fileAddr string) (*Manager, error) {
	userConn, err := grpc.NewClient(userAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user-service: %w", err)
	}
	logger.Info("Admin: connected to user-service", zap.String("addr", userAddr))

	groupConn, err := grpc.NewClient(groupAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		userConn.Close()
		return nil, fmt.Errorf("failed to connect to group-service: %w", err)
	}
	logger.Info("Admin: connected to group-service", zap.String("addr", groupAddr))

	fileConn, err := grpc.NewClient(fileAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		userConn.Close()
		groupConn.Close()
		return nil, fmt.Errorf("failed to connect to file-service: %w", err)
	}
	logger.Info("Admin: connected to file-service", zap.String("addr", fileAddr))

	return &Manager{
		userConn:    userConn,
		groupConn:   groupConn,
		fileConn:    fileConn,
		UserClient:  userpb.NewUserServiceClient(userConn),
		GroupClient: grouppb.NewGroupServiceClient(groupConn),
		FileClient:  filepb.NewFileServiceClient(fileConn),
	}, nil
}

// Close 关闭所有连接
func (m *Manager) Close() {
	if m.userConn != nil {
		m.userConn.Close()
	}
	if m.groupConn != nil {
		m.groupConn.Close()
	}
	if m.fileConn != nil {
		m.fileConn.Close()
	}
}
