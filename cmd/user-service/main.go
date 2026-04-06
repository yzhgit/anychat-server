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

	userpb "github.com/anychat/server/api/proto/user"
	authrepository "github.com/anychat/server/internal/auth/repository"
	authservice "github.com/anychat/server/internal/auth/service"
	usergrpc "github.com/anychat/server/internal/user/grpc"
	"github.com/anychat/server/internal/user/repository"
	"github.com/anychat/server/internal/user/service"
	"github.com/anychat/server/pkg/config"
	"github.com/anychat/server/pkg/database"
	grpcpkg "github.com/anychat/server/pkg/grpc"
	"github.com/anychat/server/pkg/logger"
	pkgredis "github.com/anychat/server/pkg/redis"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

const (
	serviceName = "user-service"
	version     = "v1.0.0"
)

func main() {
	fmt.Printf("Starting %s %s...\n", serviceName, version)

	// 加载配置
	if err := loadConfig(); err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// 初始化日志
	if err := initLogger(); err != nil {
		panic(fmt.Sprintf("Failed to init logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting user-service", zap.String("version", version))

	// 连接数据库
	db, err := initDatabase()
	if err != nil {
		logger.Fatal("Failed to connect database", zap.Error(err))
	}
	logger.Info("Database connected successfully")

	redisClient, err := initRedis()
	if err != nil {
		logger.Fatal("Failed to connect redis", zap.Error(err))
	}
	defer redisClient.Close()
	logger.Info("Redis connected successfully")

	// 初始化仓库
	profileRepo := repository.NewUserProfileRepository(db)
	settingsRepo := repository.NewUserSettingsRepository(db)
	qrcodeRepo := repository.NewUserQRCodeRepository(db)
	pushTokenRepo := repository.NewUserPushTokenRepository(db)
	authUserRepo := authrepository.NewUserRepository(db)
	authSessionRepo := authrepository.NewUserSessionRepository(db)
	verifyCodeRepo := authrepository.NewVerificationCodeRepository(db)

	verifyService := authservice.NewVerificationService(
		verifyCodeRepo,
		nil,
		redisClient,
		nil,
		nil,
		authservice.Config{
			AppMode:         viper.GetString("server.mode"),
			HashSecret:      viper.GetString("verify.code.hash_secret"),
			CodeLength:      viper.GetInt("verify.code.length"),
			ExpireSeconds:   viper.GetInt("verify.code.expire_seconds"),
			MaxAttempts:     viper.GetInt("verify.code.max_attempts"),
			TargetPerMinute: viper.GetInt("verify.rate_limit.target_per_minute"),
			TargetPerDay:    viper.GetInt("verify.rate_limit.target_per_day"),
			IPPerHour:       viper.GetInt("verify.rate_limit.ip_per_hour"),
			DevicePerDay:    viper.GetInt("verify.rate_limit.device_per_day"),
			DebugFixedCode:  viper.GetString("verify.code.debug_fixed_code"),
			AllowDevBypass:  viper.GetBool("verify.code.allow_dev_bypass"),
		},
	)

	// 初始化服务
	userService := service.NewUserService(
		profileRepo,
		settingsRepo,
		qrcodeRepo,
		pushTokenRepo,
		authUserRepo,
		authSessionRepo,
		verifyService,
	)

	// 初始化gRPC服务器
	grpcServer := initGRPCServer(userService)

	// 启动gRPC服务器
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

	// 初始化简化的HTTP服务器（仅健康检查）
	httpServer := initHTTPServer()

	// 启动HTTP服务器
	go func() {
		addr := fmt.Sprintf(":%d", viper.GetInt("server.http_port"))
		logger.Info("HTTP server listening (health check only)", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	logger.Info("User service started successfully")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gracefully...")

	// 关闭gRPC服务器
	grpcServer.GracefulStop()

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// 关闭数据库
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}

	logger.Info("Service stopped!")
}

// loadConfig 加载配置
func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("server.http_port", 8002)
	viper.SetDefault("server.grpc_port", 9002)
	viper.SetDefault("database.postgres.host", "localhost")
	viper.SetDefault("database.postgres.port", 5432)
	viper.SetDefault("database.postgres.user", "anychat")
	viper.SetDefault("database.postgres.password", "anychat123")
	viper.SetDefault("database.postgres.database", "anychat")
	viper.SetDefault("database.redis.host", "localhost")
	viper.SetDefault("database.redis.port", 6379)
	viper.SetDefault("database.redis.password", "")
	viper.SetDefault("database.redis.db", 0)
	viper.SetDefault("database.redis.pool_size", 10)
	viper.SetDefault("jwt.secret", "your-secret-key-change-in-production")
	viper.SetDefault("jwt.access_token_expire", 7200)
	viper.SetDefault("jwt.refresh_token_expire", 604800)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("services.auth.grpc_addr", "localhost:9001")
	viper.SetDefault("services.user.grpc_addr", "localhost:9002")
	viper.SetDefault("server.mode", "development")
	viper.SetDefault("verify.code.length", 6)
	viper.SetDefault("verify.code.expire_seconds", 300)
	viper.SetDefault("verify.code.max_attempts", 5)
	viper.SetDefault("verify.code.hash_secret", "change-me-for-production")
	viper.SetDefault("verify.code.debug_fixed_code", "123456")
	viper.SetDefault("verify.code.allow_dev_bypass", true)
	viper.SetDefault("verify.rate_limit.target_per_minute", 1)
	viper.SetDefault("verify.rate_limit.target_per_day", 10)
	viper.SetDefault("verify.rate_limit.ip_per_hour", 200)
	viper.SetDefault("verify.rate_limit.device_per_day", 100)

	// 自动读取环境变量
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		// 配置文件不存在，使用默认值
		fmt.Println("Config file not found, using defaults")
	}
	config.ExpandEnvInConfig()

	return nil
}

// initLogger 初始化日志
func initLogger() error {
	return logger.Init(&logger.Config{
		Level:    viper.GetString("log.level"),
		Output:   viper.GetString("log.output"),
		FilePath: viper.GetString("log.file_path"),
	})
}

// initDatabase 初始化数据库
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

func initRedis() (*pkgredis.Client, error) {
	return pkgredis.NewClient(&pkgredis.Config{
		Host:     viper.GetString("database.redis.host"),
		Port:     viper.GetInt("database.redis.port"),
		Password: viper.GetString("database.redis.password"),
		DB:       viper.GetInt("database.redis.db"),
		PoolSize: viper.GetInt("database.redis.pool_size"),
	})
}

// initGRPCServer 初始化gRPC服务器
func initGRPCServer(userService service.UserService) *grpc.Server {
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcpkg.RecoveryInterceptor(),
			grpcpkg.LoggingInterceptor(),
		),
	)

	userpb.RegisterUserServiceServer(grpcServer, usergrpc.NewUserServer(userService))

	return grpcServer
}

// initHTTPServer 初始化HTTP服务器（仅健康检查）
func initHTTPServer() *http.Server {
	// 设置Gin模式
	if viper.GetString("server.mode") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	r := gin.New()

	// 健康检查接口
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
