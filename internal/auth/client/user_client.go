package client

import (
	"context"
	"fmt"

	userpb "github.com/anychat/server/api/proto/user"
	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// UserClient user服务gRPC客户端
type UserClient struct {
	conn   *grpc.ClientConn
	client userpb.UserServiceClient
}

// NewUserClient 创建user客户端
func NewUserClient(addr string) (*UserClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}

	logger.Info("Connected to user-service", zap.String("addr", addr))

	return &UserClient{
		conn:   conn,
		client: userpb.NewUserServiceClient(conn),
	}, nil
}

// Close 关闭连接
func (c *UserClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// InitUserData 初始化用户数据（注册后调用）
func (c *UserClient) InitUserData(ctx context.Context, userID, nickname string) error {
	req := &userpb.InitUserDataRequest{
		UserId:   userID,
		Nickname: nickname,
	}

	_, err := c.client.InitUserData(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to init user data: %w", err)
	}

	return nil
}
