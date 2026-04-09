package client

import (
	"fmt"

	authpb "github.com/anychat/server/api/proto/auth"
	callingpb "github.com/anychat/server/api/proto/calling"
	conversationpb "github.com/anychat/server/api/proto/conversation"
	filepb "github.com/anychat/server/api/proto/file"
	friendpb "github.com/anychat/server/api/proto/friend"
	grouppb "github.com/anychat/server/api/proto/group"
	messagepb "github.com/anychat/server/api/proto/message"
	syncpb "github.com/anychat/server/api/proto/sync"
	userpb "github.com/anychat/server/api/proto/user"
	versionpb "github.com/anychat/server/api/proto/version"
	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Manager gRPC client manager
type Manager struct {
	authConn           *grpc.ClientConn
	userConn           *grpc.ClientConn
	friendConn         *grpc.ClientConn
	groupConn          *grpc.ClientConn
	fileConn           *grpc.ClientConn
	messageConn        *grpc.ClientConn
	conversationConn   *grpc.ClientConn
	syncConn           *grpc.ClientConn
	callingConn        *grpc.ClientConn
	versionConn        *grpc.ClientConn
	authClient         authpb.AuthServiceClient
	userClient         userpb.UserServiceClient
	friendClient       friendpb.FriendServiceClient
	groupClient        grouppb.GroupServiceClient
	fileClient         filepb.FileServiceClient
	messageClient      messagepb.MessageServiceClient
	conversationClient conversationpb.ConversationServiceClient
	syncClient         syncpb.SyncServiceClient
	callingClient      callingpb.CallingServiceClient
	versionClient      versionpb.VersionServiceClient
}

// NewManager creates gRPC client manager
func NewManager(authAddr, userAddr, friendAddr, groupAddr, fileAddr, messageAddr, conversationAddr, syncAddr, callingAddr, versionAddr string) (*Manager, error) {
	// Connect to auth-service
	authConn, err := grpc.NewClient(
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth-service: %w", err)
	}
	logger.Info("Connected to auth-service", zap.String("addr", authAddr))

	// Connect to user-service
	userConn, err := grpc.NewClient(
		userAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		return nil, fmt.Errorf("failed to connect to user-service: %w", err)
	}
	logger.Info("Connected to user-service", zap.String("addr", userAddr))

	// Connect to friend-service
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

	// Connect to group-service
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

	// Connect to file-service
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

	// Connect to message-service
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

	// Connect to conversation-service
	conversationConn, err := grpc.NewClient(
		conversationAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		userConn.Close()
		friendConn.Close()
		groupConn.Close()
		fileConn.Close()
		messageConn.Close()
		return nil, fmt.Errorf("failed to connect to conversation-service: %w", err)
	}
	logger.Info("Connected to conversation-service", zap.String("addr", conversationAddr))

	// Connect to sync-service
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
		conversationConn.Close()
		return nil, fmt.Errorf("failed to connect to sync-service: %w", err)
	}
	logger.Info("Connected to sync-service", zap.String("addr", syncAddr))

	// Connect to calling-service
	callingConn, err := grpc.NewClient(
		callingAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		userConn.Close()
		friendConn.Close()
		groupConn.Close()
		fileConn.Close()
		messageConn.Close()
		conversationConn.Close()
		syncConn.Close()
		return nil, fmt.Errorf("failed to connect to calling-service: %w", err)
	}
	logger.Info("Connected to calling-service", zap.String("addr", callingAddr))

	// Connect to version-service
	versionConn, err := grpc.NewClient(
		versionAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		authConn.Close()
		userConn.Close()
		friendConn.Close()
		groupConn.Close()
		fileConn.Close()
		messageConn.Close()
		conversationConn.Close()
		syncConn.Close()
		callingConn.Close()
		return nil, fmt.Errorf("failed to connect to version-service: %w", err)
	}
	logger.Info("Connected to version-service", zap.String("addr", versionAddr))

	return &Manager{
		authConn:           authConn,
		userConn:           userConn,
		friendConn:         friendConn,
		groupConn:          groupConn,
		fileConn:           fileConn,
		messageConn:        messageConn,
		conversationConn:   conversationConn,
		syncConn:           syncConn,
		callingConn:        callingConn,
		versionConn:        versionConn,
		authClient:         authpb.NewAuthServiceClient(authConn),
		userClient:         userpb.NewUserServiceClient(userConn),
		friendClient:       friendpb.NewFriendServiceClient(friendConn),
		groupClient:        grouppb.NewGroupServiceClient(groupConn),
		fileClient:         filepb.NewFileServiceClient(fileConn),
		messageClient:      messagepb.NewMessageServiceClient(messageConn),
		conversationClient: conversationpb.NewConversationServiceClient(conversationConn),
		syncClient:         syncpb.NewSyncServiceClient(syncConn),
		callingClient:      callingpb.NewCallingServiceClient(callingConn),
		versionClient:      versionpb.NewVersionServiceClient(versionConn),
	}, nil
}

// Auth get auth service client
func (m *Manager) Auth() authpb.AuthServiceClient {
	return m.authClient
}

// User get user service client
func (m *Manager) User() userpb.UserServiceClient {
	return m.userClient
}

// Friend get friend service client
func (m *Manager) Friend() friendpb.FriendServiceClient {
	return m.friendClient
}

// Group get group service client
func (m *Manager) Group() grouppb.GroupServiceClient {
	return m.groupClient
}

// File get file service client
func (m *Manager) File() filepb.FileServiceClient {
	return m.fileClient
}

// Message get message service client
func (m *Manager) Message() messagepb.MessageServiceClient {
	return m.messageClient
}

// Conversation get conversation service client
func (m *Manager) Conversation() conversationpb.ConversationServiceClient {
	return m.conversationClient
}

// Sync get sync service client
func (m *Manager) Sync() syncpb.SyncServiceClient {
	return m.syncClient
}

// Calling get audio/video call service client
func (m *Manager) Calling() callingpb.CallingServiceClient {
	return m.callingClient
}

// Version get version service client
func (m *Manager) Version() versionpb.VersionServiceClient {
	return m.versionClient
}

// Close close all connections
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

	if m.conversationConn != nil {
		if err := m.conversationConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close conversation connection: %w", err))
		}
	}

	if m.syncConn != nil {
		if err := m.syncConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close sync connection: %w", err))
		}
	}

	if m.callingConn != nil {
		if err := m.callingConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close calling connection: %w", err))
		}
	}

	if m.versionConn != nil {
		if err := m.versionConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close version connection: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}
