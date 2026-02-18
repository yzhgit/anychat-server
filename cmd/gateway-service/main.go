package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anychat/server/internal/gateway/client"
	"github.com/anychat/server/internal/gateway/handler"
	gwnotification "github.com/anychat/server/internal/gateway/notification"
	gwwebsocket "github.com/anychat/server/internal/gateway/websocket"
	"github.com/anychat/server/pkg/jwt"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "github.com/anychat/server/docs/api/swagger" // Import swagger docs
	"github.com/anychat/server/pkg/config"
)

// @title           AnyChat Gateway API
// @version         1.0
// @description     AnyChat 即时通讯系统的网关 API 服务，提供用户认证、用户管理等功能的 HTTP 接口。
// @description     所有需要认证的接口都需要在 Header 中携带 Authorization: Bearer <token>

// @contact.name   AnyChat API Support
// @contact.url    https://github.com/yzhgit/anychat-server
// @contact.email  support@anychat.example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

const (
	serviceName = "gateway-service"
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

	logger.Info("Starting gateway-service", zap.String("version", version))

	// 初始化JWT管理器
	jwtManager := initJWT()

	// 连接到后端gRPC服务
	clientManager, err := client.NewManager(
		viper.GetString("services.auth.grpc_addr"),
		viper.GetString("services.user.grpc_addr"),
		viper.GetString("services.friend.grpc_addr"),
		viper.GetString("services.group.grpc_addr"),
		viper.GetString("services.file.grpc_addr"),
		viper.GetString("services.message.grpc_addr"),
		viper.GetString("services.session.grpc_addr"),
		viper.GetString("services.sync.grpc_addr"),
		viper.GetString("services.rtc.grpc_addr"),
	)
	if err != nil {
		logger.Fatal("Failed to connect to backend services", zap.Error(err))
	}
	defer clientManager.Close()

	logger.Info("Connected to all backend services")

	// 连接NATS
	nc, err := connectNATS()
	if err != nil {
		logger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer nc.Close()
	logger.Info("Connected to NATS")

	// 初始化WebSocket管理器
	wsManager := gwwebsocket.NewManager()

	// 初始化通知订阅器
	subscriber := gwnotification.NewSubscriber(nc, wsManager)

	// 初始化HTTP服务器
	httpServer := initHTTPServer(clientManager, jwtManager, wsManager, subscriber)

	// 启动HTTP服务器
	go func() {
		addr := fmt.Sprintf(":%d", viper.GetInt("gateway.http_port"))
		logger.Info("Gateway HTTP server listening", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	logger.Info("Gateway service started successfully")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gracefully...")

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
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
	viper.SetDefault("gateway.http_port", 8080)
	viper.SetDefault("services.auth.grpc_addr", "localhost:9001")
	viper.SetDefault("services.user.grpc_addr", "localhost:9002")
	viper.SetDefault("services.friend.grpc_addr", "localhost:9003")
	viper.SetDefault("services.group.grpc_addr", "localhost:9004")
	viper.SetDefault("services.file.grpc_addr", "localhost:9007")
	viper.SetDefault("services.message.grpc_addr", "localhost:9005")
	viper.SetDefault("services.session.grpc_addr", "localhost:9006")
	viper.SetDefault("services.sync.grpc_addr", "localhost:9010")
	viper.SetDefault("services.rtc.grpc_addr", "localhost:9009")
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("jwt.secret", "your-secret-key-change-in-production")
	viper.SetDefault("jwt.access_token_expire", 7200)
	viper.SetDefault("jwt.refresh_token_expire", 604800)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("server.mode", "development")

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

// initJWT 初始化JWT管理器
func initJWT() *jwt.Manager {
	return jwt.NewManager(&jwt.Config{
		Secret:             viper.GetString("jwt.secret"),
		AccessTokenExpire:  time.Duration(viper.GetInt("jwt.access_token_expire")) * time.Second,
		RefreshTokenExpire: time.Duration(viper.GetInt("jwt.refresh_token_expire")) * time.Second,
	})
}

// connectNATS 连接到NATS
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

// initHTTPServer 初始化HTTP服务器
func initHTTPServer(clientManager *client.Manager, jwtManager *jwt.Manager,
	wsManager *gwwebsocket.Manager, subscriber *gwnotification.Subscriber) *http.Server {
	// 设置Gin模式
	if viper.GetString("server.mode") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	r := gin.New()

	// 中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Swagger文档路由（仅在非生产环境）
	if viper.GetString("server.mode") != "release" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// 注册路由
	handler.RegisterRoutes(r, clientManager, jwtManager, wsManager, subscriber)

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", viper.GetInt("gateway.http_port")),
		Handler: r,
	}
}
