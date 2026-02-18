package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	friendpb "github.com/anychat/server/api/proto/friend"
	grouppb "github.com/anychat/server/api/proto/group"
	messagepb "github.com/anychat/server/api/proto/message"
	sessionpb "github.com/anychat/server/api/proto/session"
	syncpb "github.com/anychat/server/api/proto/sync"
	syncgrpc "github.com/anychat/server/internal/sync/grpc"
	"github.com/anychat/server/internal/sync/service"
	grpcpkg "github.com/anychat/server/pkg/grpc"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"github.com/anychat/server/pkg/config"
)

const (
	serviceName = "sync-service"
	version     = "v1.0.0"
)

func main() {
	fmt.Printf("Starting %s %s...\n", serviceName, version)

	if err := loadConfig(); err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	if err := initLogger(); err != nil {
		panic(fmt.Sprintf("Failed to init logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting sync-service", zap.String("version", version))

	// 连接上游服务
	friendClient, err := connectService[friendpb.FriendServiceClient](
		viper.GetString("services.friend.grpc_addr"), "friend-service",
		func(cc *grpc.ClientConn) friendpb.FriendServiceClient {
			return friendpb.NewFriendServiceClient(cc)
		})
	if err != nil {
		logger.Fatal("Failed to connect friend-service", zap.Error(err))
	}

	groupClient, err := connectService[grouppb.GroupServiceClient](
		viper.GetString("services.group.grpc_addr"), "group-service",
		func(cc *grpc.ClientConn) grouppb.GroupServiceClient {
			return grouppb.NewGroupServiceClient(cc)
		})
	if err != nil {
		logger.Fatal("Failed to connect group-service", zap.Error(err))
	}

	sessionClient, err := connectService[sessionpb.SessionServiceClient](
		viper.GetString("services.session.grpc_addr"), "session-service",
		func(cc *grpc.ClientConn) sessionpb.SessionServiceClient {
			return sessionpb.NewSessionServiceClient(cc)
		})
	if err != nil {
		logger.Fatal("Failed to connect session-service", zap.Error(err))
	}

	messageClient, err := connectService[messagepb.MessageServiceClient](
		viper.GetString("services.message.grpc_addr"), "message-service",
		func(cc *grpc.ClientConn) messagepb.MessageServiceClient {
			return messagepb.NewMessageServiceClient(cc)
		})
	if err != nil {
		logger.Fatal("Failed to connect message-service", zap.Error(err))
	}

	logger.Info("Connected to all upstream services")

	// 连接NATS
	nc, err := connectNATS()
	if err != nil {
		logger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer nc.Close()
	logger.Info("Connected to NATS")

	notificationPub := notification.NewPublisher(nc)

	// 初始化服务
	syncSvc := service.NewSyncService(friendClient, groupClient, sessionClient, messageClient, notificationPub)

	// 初始化并启动gRPC服务器
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcpkg.RecoveryInterceptor(),
			grpcpkg.LoggingInterceptor(),
		),
	)
	syncpb.RegisterSyncServiceServer(grpcServer, syncgrpc.NewServer(syncSvc))

	go func() {
		grpcPort := viper.GetInt("server.grpc_port")
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
		if err != nil {
			logger.Fatal("Failed to listen gRPC", zap.Error(err))
		}
		logger.Info("gRPC server listening", zap.Int("port", grpcPort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// 启动健康检查HTTP服务器
	httpServer := initHTTPServer()
	go func() {
		addr := fmt.Sprintf(":%d", viper.GetInt("server.http_port"))
		logger.Info("HTTP server listening", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	logger.Info("Sync service started successfully")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gracefully...")
	grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	logger.Info("Service stopped!")
}

// connectService 泛型辅助：建立 gRPC 连接并返回客户端
func connectService[T any](addr, name string, newClient func(*grpc.ClientConn) T) (T, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		var zero T
		return zero, fmt.Errorf("failed to connect to %s at %s: %w", name, addr, err)
	}
	logger.Info("Connected to "+name, zap.String("addr", addr))
	return newClient(conn), nil
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	viper.SetDefault("server.http_port", 8010)
	viper.SetDefault("server.grpc_port", 9010)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("services.friend.grpc_addr", "localhost:9003")
	viper.SetDefault("services.group.grpc_addr", "localhost:9005")
	viper.SetDefault("services.session.grpc_addr", "localhost:9006")
	viper.SetDefault("services.message.grpc_addr", "localhost:9004")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		fmt.Println("Config file not found, using defaults")
	}
	config.ExpandEnvInConfig()
	return nil
}

func initLogger() error {
	return logger.Init(&logger.Config{
		Level:    viper.GetString("log.level"),
		Output:   viper.GetString("log.output"),
		FilePath: viper.GetString("log.file_path"),
	})
}

func connectNATS() (*nats.Conn, error) {
	natsURL := viper.GetString("nats.url")
	return nats.Connect(natsURL,
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Warn("NATS connection closed")
		}),
	)
}

func initHTTPServer() *http.Server {
	if viper.GetString("server.mode") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": serviceName,
			"version": version,
		})
	})
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", viper.GetInt("server.http_port")),
		Handler: r,
	}
}
