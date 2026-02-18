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

	rtcpb "github.com/anychat/server/api/proto/rtc"
	rtcgrpc "github.com/anychat/server/internal/rtc/grpc"
	"github.com/anychat/server/internal/rtc/repository"
	"github.com/anychat/server/internal/rtc/service"
	"github.com/anychat/server/pkg/database"
	grpcpkg "github.com/anychat/server/pkg/grpc"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"github.com/anychat/server/pkg/config"
)

const (
	serviceName = "rtc-service"
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

	logger.Info("Starting rtc-service", zap.String("version", version))

	// 连接数据库
	db, err := initDatabase()
	if err != nil {
		logger.Fatal("Failed to connect database", zap.Error(err))
	}
	logger.Info("Database connected successfully")

	// 连接 NATS
	nc, err := connectNATS()
	if err != nil {
		logger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer nc.Close()
	logger.Info("Connected to NATS")

	// 初始化通知发布器
	notificationPub := notification.NewPublisher(nc)

	// 初始化仓库
	callRepo := repository.NewCallRepository(db)
	meetingRepo := repository.NewMeetingRepository(db)

	// 初始化 LiveKit 服务
	lkSvc := service.NewRTCService(
		viper.GetString("livekit.url"),
		viper.GetString("livekit.api_key"),
		viper.GetString("livekit.api_secret"),
		callRepo,
		meetingRepo,
		notificationPub,
	)

	// 初始化并启动 gRPC 服务器
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcpkg.RecoveryInterceptor(),
			grpcpkg.LoggingInterceptor(),
		),
	)
	rtcpb.RegisterRTCServiceServer(grpcServer, rtcgrpc.NewServer(lkSvc))

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

	// 健康检查 HTTP 服务器
	httpServer := initHTTPServer()
	go func() {
		addr := fmt.Sprintf(":%d", viper.GetInt("server.http_port"))
		logger.Info("HTTP server listening", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	logger.Info("RTC service started successfully")

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

	viper.SetDefault("server.http_port", 8009)
	viper.SetDefault("server.grpc_port", 9009)
	viper.SetDefault("database.postgres.host", "localhost")
	viper.SetDefault("database.postgres.port", 5432)
	viper.SetDefault("database.postgres.user", "anychat")
	viper.SetDefault("database.postgres.password", "anychat123")
	viper.SetDefault("database.postgres.database", "anychat")
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("livekit.url", "ws://localhost:7880")
	viper.SetDefault("livekit.api_key", "devkey")
	viper.SetDefault("livekit.api_secret", "secret")
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
