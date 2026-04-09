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

	grouppb "github.com/anychat/server/api/proto/group"
	messagepb "github.com/anychat/server/api/proto/message"
	userpb "github.com/anychat/server/api/proto/user"
	groupgrpc "github.com/anychat/server/internal/group/grpc"
	"github.com/anychat/server/internal/group/repository"
	"github.com/anychat/server/internal/group/service"
	"github.com/anychat/server/pkg/config"
	"github.com/anychat/server/pkg/database"
	grpcpkg "github.com/anychat/server/pkg/grpc"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

const (
	serviceName = "group-service"
	version     = "v1.0.0"
)

func main() {
	fmt.Printf("Starting %s %s...\n", serviceName, version)

	// Load config
	if err := loadConfig(); err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	if err := initLogger(); err != nil {
		panic(fmt.Sprintf("Failed to init logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting group-service", zap.String("version", version))

	// Connect to database
	db, err := initDatabase()
	if err != nil {
		logger.Fatal("Failed to connect database", zap.Error(err))
	}
	logger.Info("Database connected successfully")

	// Connect to user-service
	userClient, err := connectUserService()
	if err != nil {
		logger.Fatal("Failed to connect to user-service", zap.Error(err))
	}
	logger.Info("Connected to user-service")

	// Connect to message-service
	messageClient, err := connectMessageService()
	if err != nil {
		logger.Fatal("Failed to connect to message-service", zap.Error(err))
	}
	logger.Info("Connected to message-service")

	// Connect to NATS
	nc, err := connectNATS()
	if err != nil {
		logger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	logger.Info("Connected to NATS")

	// Initialize notification publisher
	notificationPub := notification.NewPublisher(nc)

	// Initialize repositories
	groupRepo := repository.NewGroupRepository(db)
	memberRepo := repository.NewGroupMemberRepository(db)
	settingRepo := repository.NewGroupSettingRepository(db)
	joinRequestRepo := repository.NewGroupJoinRequestRepository(db)
	pinnedRepo := repository.NewGroupPinnedMessageRepository(db)
	qrcodeRepo := repository.NewGroupQRCodeRepository(db)

	// Initialize services
	groupService := service.NewGroupService(groupRepo, memberRepo, settingRepo, joinRequestRepo, pinnedRepo, qrcodeRepo, messageClient, userClient, notificationPub, db)

	// Initialize gRPC server
	grpcServer := initGRPCServer(groupService)

	// Start gRPC server
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

	// Initialize simplified HTTP server (health check only)
	httpServer := initHTTPServer()

	// Start HTTP server
	go func() {
		addr := fmt.Sprintf(":%d", viper.GetInt("server.http_port"))
		logger.Info("HTTP server listening (health check only)", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	logger.Info("Group service started successfully")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gracefully...")

	// Stop gRPC server
	grpcServer.GracefulStop()

	// Stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Close NATS connection
	if nc != nil {
		nc.Close()
	}

	// Close database
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}

	logger.Info("Service stopped!")
}

// loadConfig loads configuration
func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set default values
	viper.SetDefault("server.http_port", 8004)
	viper.SetDefault("server.grpc_port", 9004)
	viper.SetDefault("database.postgres.host", "localhost")
	viper.SetDefault("database.postgres.port", 5432)
	viper.SetDefault("database.postgres.user", "anychat")
	viper.SetDefault("database.postgres.password", "anychat123")
	viper.SetDefault("database.postgres.database", "anychat")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("services.user.grpc_addr", "localhost:9002")
	viper.SetDefault("services.message.grpc_addr", "localhost:9005")
	viper.SetDefault("services.group.grpc_addr", "localhost:9004")

	// Auto-read environment variables
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		// Config file not found, use defaults
		fmt.Println("Config file not found, using defaults")
	}
	config.ExpandEnvInConfig()

	return nil
}

// initLogger initializes logger
func initLogger() error {
	return logger.Init(&logger.Config{
		Level:    viper.GetString("log.level"),
		Output:   viper.GetString("log.output"),
		FilePath: viper.GetString("log.file_path"),
	})
}

// initDatabase initializes database
func initDatabase() (*gorm.DB, error) {
	logLevel := gormLogger.Silent
	if viper.GetString("log.level") == "debug" {
		logLevel = gormLogger.Info
	}

	return database.NewPostgresDB(&database.Config{
		Host:            viper.GetString("database.postgres.host"),
		Port:            viper.GetInt("database.postgres.port"),
		User:            viper.GetString("database.postgres.user"),
		Password:        viper.GetString("database.postgres.password"),
		DBName:          viper.GetString("database.postgres.database"),
		MaxOpenConns:    viper.GetInt("database.postgres.max_open_conns"),
		MaxIdleConns:    viper.GetInt("database.postgres.max_idle_conns"),
		ConnMaxLifetime: viper.GetInt("database.postgres.conn_max_lifetime"),
		LogLevel:        logLevel,
	})
}

// connectUserService connects to user-service
func connectUserService() (userpb.UserServiceClient, error) {
	addr := viper.GetString("services.user.grpc_addr")
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}

	return userpb.NewUserServiceClient(conn), nil
}

// connectMessageService connects to message-service
func connectMessageService() (messagepb.MessageServiceClient, error) {
	addr := viper.GetString("services.message.grpc_addr")
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to message service: %w", err)
	}

	return messagepb.NewMessageServiceClient(conn), nil
}

// connectNATS connects to NATS
func connectNATS() (*nats.Conn, error) {
	natsURL := viper.GetString("nats.url")
	nc, err := nats.Connect(natsURL,
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
	return nc, err
}

// initGRPCServer initializes gRPC server
func initGRPCServer(groupService service.GroupService) *grpc.Server {
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcpkg.RecoveryInterceptor(),
			grpcpkg.LoggingInterceptor(),
		),
	)

	grouppb.RegisterGroupServiceServer(grpcServer, groupgrpc.NewGroupServer(groupService))

	return grpcServer
}

// initHTTPServer initializes HTTP server (health check only)
func initHTTPServer() *http.Server {
	// Set Gin mode
	if viper.GetString("server.mode") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	r := gin.New()

	// Health check endpoint
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
