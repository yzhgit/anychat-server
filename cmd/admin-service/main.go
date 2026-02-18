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

	adminpb "github.com/anychat/server/api/proto/admin"
	adminclient "github.com/anychat/server/internal/admin/client"
	admingrpc "github.com/anychat/server/internal/admin/grpc"
	adminhandler "github.com/anychat/server/internal/admin/handler"
	"github.com/anychat/server/internal/admin/repository"
	"github.com/anychat/server/internal/admin/service"
	"github.com/anychat/server/pkg/database"
	grpcpkg "github.com/anychat/server/pkg/grpc"
	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"github.com/anychat/server/pkg/config"
)

const (
	serviceName = "admin-service"
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

	logger.Info("Starting admin-service", zap.String("version", version))

	// 连接数据库
	db, err := initDatabase()
	if err != nil {
		logger.Fatal("Failed to connect database", zap.Error(err))
	}
	logger.Info("Database connected successfully")

	// 初始化仓库
	adminRepo := repository.NewAdminUserRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)
	configRepo := repository.NewSystemConfigRepository(db)

	// 连接下游服务
	clientManager, err := adminclient.NewManager(
		viper.GetString("services.user.grpc_addr"),
		viper.GetString("services.group.grpc_addr"),
	)
	if err != nil {
		logger.Fatal("Failed to connect to backend services", zap.Error(err))
	}
	defer clientManager.Close()

	// JWT管理器（管理员专用独立secret）
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret:            viper.GetString("admin.jwt.secret"),
		AccessTokenExpire: time.Duration(viper.GetInt("admin.jwt.access_token_expire")) * time.Second,
	})

	// 初始化业务服务
	adminSvc := service.NewAdminService(
		jwtManager,
		adminRepo,
		auditRepo,
		configRepo,
		clientManager.UserClient,
		clientManager.GroupClient,
	)

	// 初始化gRPC服务器
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcpkg.RecoveryInterceptor(),
			grpcpkg.LoggingInterceptor(),
		),
	)
	adminpb.RegisterAdminServiceServer(grpcServer, admingrpc.NewServer(adminSvc))

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

	// 初始化HTTP服务器
	if viper.GetString("server.mode") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	adminhandler.RegisterRoutes(r, adminSvc, jwtManager)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", viper.GetInt("server.http_port")),
		Handler: r,
	}

	go func() {
		logger.Info("HTTP server listening", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	logger.Info("Admin service started successfully")

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

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}
	logger.Info("Service stopped!")
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	viper.SetDefault("server.http_port", 8011)
	viper.SetDefault("server.grpc_port", 9011)
	viper.SetDefault("server.mode", "development")
	viper.SetDefault("database.postgres.host", "localhost")
	viper.SetDefault("database.postgres.port", 5432)
	viper.SetDefault("database.postgres.user", "anychat")
	viper.SetDefault("database.postgres.password", "anychat123")
	viper.SetDefault("database.postgres.database", "anychat")
	viper.SetDefault("services.user.grpc_addr", "localhost:9002")
	viper.SetDefault("services.group.grpc_addr", "localhost:9004")
	viper.SetDefault("admin.jwt.secret", "admin-secret-change-in-production")
	viper.SetDefault("admin.jwt.access_token_expire", 28800)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.output", "stdout")

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

func initDatabase() (*gorm.DB, error) {
	logLevel := gormLogger.Silent
	if viper.GetString("log.level") == "debug" {
		logLevel = gormLogger.Info
	}
	return database.NewPostgresDB(&database.Config{
		Host:     viper.GetString("database.postgres.host"),
		Port:     viper.GetInt("database.postgres.port"),
		User:     viper.GetString("database.postgres.user"),
		Password: viper.GetString("database.postgres.password"),
		DBName:   viper.GetString("database.postgres.database"),
		LogLevel: logLevel,
	})
}
