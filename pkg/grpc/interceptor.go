package grpc

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/anychat/server/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor 日志拦截器，记录所有gRPC请求
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		startTime := time.Now()

		// 调用实际的处理方法
		resp, err := handler(ctx, req)

		// 记录请求日志
		duration := time.Since(startTime)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("status", statusCode.String()),
		}

		if err != nil {
			logger.Error("gRPC request failed", append(fields, zap.Error(err))...)
		} else {
			logger.Info("gRPC request completed", fields...)
		}

		return resp, err
	}
}

// RecoveryInterceptor 恢复拦截器，捕获panic防止服务崩溃
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)
				err = status.Errorf(codes.Internal, "internal server error: %v", r)
			}
		}()

		return handler(ctx, req)
	}
}
