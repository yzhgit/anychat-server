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

	conversationpb "github.com/anychat/server/api/proto/conversation"
	friendpb "github.com/anychat/server/api/proto/friend"
	grouppb "github.com/anychat/server/api/proto/group"
	messagepb "github.com/anychat/server/api/proto/message"
	messagegrpc "github.com/anychat/server/internal/message/grpc"
	"github.com/anychat/server/internal/message/repository"
	"github.com/anychat/server/internal/message/service"
	"github.com/anychat/server/internal/message/worker"
	"github.com/anychat/server/pkg/config"
	"github.com/anychat/server/pkg/database"
	grpcpkg "github.com/anychat/server/pkg/grpc"
	"github.com/anychat/server/pkg/logger"
	"github.com/anychat/server/pkg/notification"
	pkgredis "github.com/anychat/server/pkg/redis"
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
	serviceName = "message-service"
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

	logger.Info("Starting message-service", zap.String("version", version))

	// Connect to database
	db, err := initDatabase()
	if err != nil {
		logger.Fatal("Failed to connect database", zap.Error(err))
	}
	logger.Info("Database connected successfully")

	// Connect to Redis
	redisClient, err := initRedis()
	if err != nil {
		logger.Fatal("Failed to connect Redis", zap.Error(err))
	}
	defer redisClient.Close()
	logger.Info("Redis connected successfully")

	// Connect to NATS
	nc, err := connectNATS()
	if err != nil {
		logger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer nc.Close()
	logger.Info("NATS connected successfully")

	// Initialize notification publisher
	notificationPub := notification.NewPublisher(nc)
	logger.Info("Notification publisher initialized")

	// Connect to dependent services
	conversationConn, conversationClient, err := connectConversationService()
	if err != nil {
		logger.Fatal("Failed to connect conversation-service", zap.Error(err))
	}
	defer conversationConn.Close()

	groupConn, groupClient, err := connectGroupService()
	if err != nil {
		logger.Fatal("Failed to connect group-service", zap.Error(err))
	}
	defer groupConn.Close()

	friendConn, friendClient, err := connectFriendService()
	if err != nil {
		logger.Fatal("Failed to connect friend-service", zap.Error(err))
	}
	defer friendConn.Close()

	// Initialize repositories
	messageRepo := repository.NewMessageRepository(db)
	readReceiptRepo := repository.NewReadReceiptRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	sendIdempotencyRepo := repository.NewSendIdempotencyRepository(db)
	typingRepo := repository.NewTypingRepository(redisClient)

	// Initialize services
	messageService := service.NewMessageService(
		messageRepo,
		readReceiptRepo,
		sequenceRepo,
		sendIdempotencyRepo,
		typingRepo,
		service.TypingConfig{
			DefaultTTL:   time.Duration(viper.GetInt("typing.default_ttl_seconds")) * time.Second,
			MinTTL:       time.Duration(viper.GetInt("typing.min_ttl_seconds")) * time.Second,
			MaxTTL:       time.Duration(viper.GetInt("typing.max_ttl_seconds")) * time.Second,
			EmitDebounce: time.Duration(viper.GetInt("typing.emit_debounce_seconds")) * time.Second,
		},
		conversationClient,
		friendClient,
		groupClient,
		notificationPub,
		db,
	)

	// Initialize and start auto delete worker
	autoDeleteWorker := worker.NewAutoDeleteWorker(
		messageRepo,
		notificationPub,
		1000,
		1*time.Minute,
	)
	autoDeleteWorker.StartAsync()
	logger.Info("AutoDeleteWorker started")

	// Initialize gRPC server
	grpcServer := initGRPCServer(messageService)

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

	logger.Info("Message service started successfully")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gracefully...")

	// Stop auto delete worker
	autoDeleteWorker.Stop()
	logger.Info("AutoDeleteWorker stopped")

	// Stop gRPC server
	grpcServer.GracefulStop()

	// Stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Close NATS connection
	nc.Close()

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
	viper.SetDefault("server.http_port", 8005)
	viper.SetDefault("server.grpc_port", 9005)
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
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("typing.default_ttl_seconds", 5)
	viper.SetDefault("typing.min_ttl_seconds", 3)
	viper.SetDefault("typing.max_ttl_seconds", 8)
	viper.SetDefault("typing.emit_debounce_seconds", 2)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("services.message.grpc_addr", "localhost:9005")
	viper.SetDefault("services.conversation.grpc_addr", "localhost:9006")
	viper.SetDefault("services.group.grpc_addr", "localhost:9004")
	viper.SetDefault("services.friend.grpc_addr", "localhost:9003")

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

func initRedis() (*pkgredis.Client, error) {
	return pkgredis.NewClient(&pkgredis.Config{
		Host:     viper.GetString("database.redis.host"),
		Port:     viper.GetInt("database.redis.port"),
		Password: viper.GetString("database.redis.password"),
		DB:       viper.GetInt("database.redis.db"),
		PoolSize: viper.GetInt("database.redis.pool_size"),
	})
}

// connectNATS connects to NATS server
func connectNATS() (*nats.Conn, error) {
	natsURL := viper.GetString("nats.url")

	nc, err := nats.Connect(
		natsURL,
		nats.Name(serviceName),
		nats.MaxReconnects(-1), // Unlimited reconnects
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Error("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Info("NATS connection closed")
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return nc, nil
}

func connectConversationService() (*grpc.ClientConn, conversationpb.ConversationServiceClient, error) {
	addr := viper.GetString("services.conversation.grpc_addr")
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect conversation service: %w", err)
	}
	return conn, conversationpb.NewConversationServiceClient(conn), nil
}

func connectGroupService() (*grpc.ClientConn, grouppb.GroupServiceClient, error) {
	addr := viper.GetString("services.group.grpc_addr")
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect group service: %w", err)
	}
	return conn, grouppb.NewGroupServiceClient(conn), nil
}

func connectFriendService() (*grpc.ClientConn, friendpb.FriendServiceClient, error) {
	addr := viper.GetString("services.friend.grpc_addr")
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect friend service: %w", err)
	}
	return conn, friendpb.NewFriendServiceClient(conn), nil
}

// initGRPCServer initializes gRPC server
func initGRPCServer(messageService service.MessageService) *grpc.Server {
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcpkg.RecoveryInterceptor(),
			grpcpkg.LoggingInterceptor(),
		),
	)

	messagepb.RegisterMessageServiceServer(grpcServer, messagegrpc.NewServer(messageService))

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
