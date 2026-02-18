package client

import (
	"fmt"

	authpb "github.com/anychat/server/api/proto/auth"
	filepb "github.com/anychat/server/api/proto/file"
	friendpb "github.com/anychat/server/api/proto/friend"
	grouppb "github.com/anychat/server/api/proto/group"
	rtcpb "github.com/anychat/server/api/proto/rtc"
	messagepb "github.com/anychat/server/api/proto/message"
	sessionpb "github.com/anychat/server/api/proto/session"
	syncpb "github.com/anychat/server/api/proto/sync"
	userpb "github.com/anychat/server/api/proto/user"
	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Manager gRPC客户端管理器
type Manager struct {
	authConn     *grpc.ClientConn
	userConn     *grpc.ClientConn
	friendConn   *grpc.ClientConn
	groupConn    *grpc.ClientConn
	fileConn     *grpc.ClientConn
	messageConn   *grpc.ClientConn
	sessionConn   *grpc.ClientConn
	syncConn      *grpc.ClientConn
	rtcConn       *grpc.ClientConn
	authClient   authpb.AuthServiceClient
	userClient   userpb.UserServiceClient
	friendClient friendpb.FriendServiceClient
	groupClient  grouppb.GroupServiceClient
	fileClient   filepb.FileServiceClient
	messageClient messagepb.MessageServiceClient
	sessionClient sessionpb.SessionServiceClient
	syncClient    syncpb.SyncServiceClient
	rtcClient     rtcpb.RTCServiceClient
}

// NewManager 创建gRPC客户端管理器
func NewManager(authAddr, userAddr, friendAddr, groupAddr, fileAddr, messageAddr, sessionAddr, syncAddr, rtcAddr string) (*Manager, error) {
	// 连接auth-service
	authConn, err := grpc.NewClient(
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth-service: %w", err)
	}
	logger.Info("Connected to auth-service", zap.String("addr", authAddr))

	// 连接user-service
	userConn, err := grpc.NewClient(
		userAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		return nil, fmt.Errorf("failed to connect to user-service: %w", err)
	}
	logger.Info("Connected to user-service", zap.String("addr", userAddr))

	// 连接friend-service
	friendConn, err := grpc.NewClient(
		friendAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		userConn.Close()
		return nil, fmt.Errorf("failed to connect to friend-service: %w", err)
	}
	logger.Info("Connected to friend-service", zap.String("addr", friendAddr))

	// 连接group-service
	groupConn, err := grpc.NewClient(
		groupAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		userConn.Close()
		friendConn.Close()
		return nil, fmt.Errorf("failed to connect to group-service: %w", err)
	}
	logger.Info("Connected to group-service", zap.String("addr", groupAddr))

	// 连接file-service
	fileConn, err := grpc.NewClient(
		fileAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		userConn.Close()
		friendConn.Close()
		groupConn.Close()
		return nil, fmt.Errorf("failed to connect to file-service: %w", err)
	}
	logger.Info("Connected to file-service", zap.String("addr", fileAddr))

	// 连接message-service
	messageConn, err := grpc.NewClient(
		messageAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		userConn.Close()
		friendConn.Close()
		groupConn.Close()
		fileConn.Close()
		return nil, fmt.Errorf("failed to connect to message-service: %w", err)
	}
	logger.Info("Connected to message-service", zap.String("addr", messageAddr))

	// 连接session-service
	sessionConn, err := grpc.NewClient(
		sessionAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		userConn.Close()
		friendConn.Close()
		groupConn.Close()
		fileConn.Close()
		messageConn.Close()
		return nil, fmt.Errorf("failed to connect to session-service: %w", err)
	}
	logger.Info("Connected to session-service", zap.String("addr", sessionAddr))

	// 连接sync-service
	syncConn, err := grpc.NewClient(
		syncAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		userConn.Close()
		friendConn.Close()
		groupConn.Close()
		fileConn.Close()
		messageConn.Close()
		sessionConn.Close()
		return nil, fmt.Errorf("failed to connect to sync-service: %w", err)
	}
	logger.Info("Connected to sync-service", zap.String("addr", syncAddr))

	// 连接rtc-service
	rtcConn, err := grpc.NewClient(
		rtcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		userConn.Close()
		friendConn.Close()
		groupConn.Close()
		fileConn.Close()
		messageConn.Close()
		sessionConn.Close()
		syncConn.Close()
		return nil, fmt.Errorf("failed to connect to rtc-service: %w", err)
	}
	logger.Info("Connected to rtc-service", zap.String("addr", rtcAddr))

	return &Manager{
		authConn:      authConn,
		userConn:      userConn,
		friendConn:    friendConn,
		groupConn:     groupConn,
		fileConn:      fileConn,
		messageConn:   messageConn,
		sessionConn:   sessionConn,
		syncConn:      syncConn,
		rtcConn:       rtcConn,
		authClient:    authpb.NewAuthServiceClient(authConn),
		userClient:    userpb.NewUserServiceClient(userConn),
		friendClient:  friendpb.NewFriendServiceClient(friendConn),
		groupClient:   grouppb.NewGroupServiceClient(groupConn),
		fileClient:    filepb.NewFileServiceClient(fileConn),
		messageClient: messagepb.NewMessageServiceClient(messageConn),
		sessionClient: sessionpb.NewSessionServiceClient(sessionConn),
		syncClient:    syncpb.NewSyncServiceClient(syncConn),
		rtcClient:     rtcpb.NewRTCServiceClient(rtcConn),
	}, nil
}

// Auth 获取auth服务客户端
func (m *Manager) Auth() authpb.AuthServiceClient {
	return m.authClient
}

// User 获取user服务客户端
func (m *Manager) User() userpb.UserServiceClient {
	return m.userClient
}

// Friend 获取friend服务客户端
func (m *Manager) Friend() friendpb.FriendServiceClient {
	return m.friendClient
}

// Group 获取group服务客户端
func (m *Manager) Group() grouppb.GroupServiceClient {
	return m.groupClient
}

// File 获取file服务客户端
func (m *Manager) File() filepb.FileServiceClient {
	return m.fileClient
}

// Message 获取message服务客户端
func (m *Manager) Message() messagepb.MessageServiceClient {
	return m.messageClient
}

// Session 获取session服务客户端
func (m *Manager) Session() sessionpb.SessionServiceClient {
	return m.sessionClient
}

// Sync 获取sync服务客户端
func (m *Manager) Sync() syncpb.SyncServiceClient {
	return m.syncClient
}

// RTC 获取rtc服务客户端
func (m *Manager) RTC() rtcpb.RTCServiceClient {
	return m.rtcClient
}

// Close 关闭所有连接
func (m *Manager) Close() error {
	var errs []error

	if m.authConn != nil {
		if err := m.authConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close auth connection: %w", err))
		}
	}

	if m.userConn != nil {
		if err := m.userConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close user connection: %w", err))
		}
	}

	if m.friendConn != nil {
		if err := m.friendConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close friend connection: %w", err))
		}
	}

	if m.groupConn != nil {
		if err := m.groupConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close group connection: %w", err))
		}
	}

	if m.fileConn != nil {
		if err := m.fileConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close file connection: %w", err))
		}
	}

	if m.messageConn != nil {
		if err := m.messageConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close message connection: %w", err))
		}
	}

	if m.sessionConn != nil {
		if err := m.sessionConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close session connection: %w", err))
		}
	}

	if m.syncConn != nil {
		if err := m.syncConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close sync connection: %w", err))
		}
	}

	if m.rtcConn != nil {
		if err := m.rtcConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close rtc connection: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}
