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
// @description     AnyChat instant messaging system gateway API service, providing HTTP interfaces for user authentication, user management, and other functions.
// @description     All endpoints requiring authentication must include Authorization: Bearer <token> in the Header.

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

	// Load config
	if err := loadConfig(); err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	if err := initLogger(); err != nil {
		panic(fmt.Sprintf("Failed to init logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting gateway-service", zap.String("version", version))

	// Initialize JWT manager
	jwtManager := initJWT()

	// Connect to backend gRPC services
	clientManager, err := client.NewManager(
		viper.GetString("services.auth.grpc_addr"),
		viper.GetString("services.user.grpc_addr"),
		viper.GetString("services.friend.grpc_addr"),
		viper.GetString("services.group.grpc_addr"),
		viper.GetString("services.file.grpc_addr"),
		viper.GetString("services.message.grpc_addr"),
		viper.GetString("services.conversation.grpc_addr"),
		viper.GetString("services.sync.grpc_addr"),
		getCallingGRPCAddr(),
		viper.GetString("services.version.grpc_addr"),
	)
	if err != nil {
		logger.Fatal("Failed to connect to backend services", zap.Error(err))
	}
	defer clientManager.Close()

	logger.Info("Connected to all backend services")

	// Connect to NATS
	nc, err := connectNATS()
	if err != nil {
		logger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer nc.Close()
	logger.Info("Connected to NATS")

	// Initialize WebSocket manager
	wsManager := gwwebsocket.NewManager()

	// Initialize notification subscriber
	subscriber := gwnotification.NewSubscriber(nc, wsManager)

	// Initialize HTTP server
	httpServer := initHTTPServer(clientManager, jwtManager, wsManager, subscriber)

	// Start HTTP server
	go func() {
		addr := fmt.Sprintf(":%d", viper.GetInt("gateway.http_port"))
		logger.Info("Gateway HTTP server listening", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	logger.Info("Gateway service started successfully")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gracefully...")

	// Stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
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
	viper.SetDefault("gateway.http_port", 8080)
	viper.SetDefault("services.auth.grpc_addr", "localhost:9001")
	viper.SetDefault("services.user.grpc_addr", "localhost:9002")
	viper.SetDefault("services.friend.grpc_addr", "localhost:9003")
	viper.SetDefault("services.group.grpc_addr", "localhost:9004")
	viper.SetDefault("services.file.grpc_addr", "localhost:9007")
	viper.SetDefault("services.message.grpc_addr", "localhost:9005")
	viper.SetDefault("services.conversation.grpc_addr", "localhost:9006")
	viper.SetDefault("services.sync.grpc_addr", "localhost:9010")
	viper.SetDefault("services.calling.grpc_addr", "localhost:9009")
	viper.SetDefault("services.version.grpc_addr", "localhost:9012")
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("jwt.secret", "your-secret-key-change-in-production")
	viper.SetDefault("jwt.access_token_expire", 7200)
	viper.SetDefault("jwt.refresh_token_expire", 604800)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("server.mode", "development")

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

func getCallingGRPCAddr() string {
	if viper.InConfig("services.calling.grpc_addr") {
		return viper.GetString("services.calling.grpc_addr")
	}
	if addr := viper.GetString("services.calling.grpc_addr"); addr != "" {
		return addr
	}
	return "localhost:9009"
}

// initLogger initializes logger
func initLogger() error {
	return logger.Init(&logger.Config{
		Level:    viper.GetString("log.level"),
		Output:   viper.GetString("log.output"),
		FilePath: viper.GetString("log.file_path"),
	})
}

// initJWT initializes JWT manager
func initJWT() *jwt.Manager {
	return jwt.NewManager(&jwt.Config{
		Secret:             viper.GetString("jwt.secret"),
		AccessTokenExpire:  time.Duration(viper.GetInt("jwt.access_token_expire")) * time.Second,
		RefreshTokenExpire: time.Duration(viper.GetInt("jwt.refresh_token_expire")) * time.Second,
	})
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

// initHTTPServer initializes HTTP server
func initHTTPServer(clientManager *client.Manager, jwtManager *jwt.Manager,
	wsManager *gwwebsocket.Manager, subscriber *gwnotification.Subscriber) *http.Server {
	// Set Gin mode
	if viper.GetString("server.mode") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	r := gin.New()

	// Middleware
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Swagger docs route (non-production only)
	if viper.GetString("server.mode") != "release" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Register routes
	handler.RegisterRoutes(r, clientManager, jwtManager, wsManager, subscriber)

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", viper.GetInt("gateway.http_port")),
		Handler: r,
	}
}
