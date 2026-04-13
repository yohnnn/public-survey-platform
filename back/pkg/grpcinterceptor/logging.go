package grpcinterceptor

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		st, _ := status.FromError(err)

		fields := []any{
			"component", "grpc",
			"method", info.FullMethod,
			"code", st.Code().String(),
			"duration_ms", float64(time.Since(start).Microseconds()) / 1000,
		}

		if requestID, ok := requestIDFromIncomingMetadata(ctx); ok {
			fields = append(fields, "request_id", requestID)
		}
		if userID, ok := UserIDFromContext(ctx); ok {
			fields = append(fields, "user_id", userID)
		}
		if err != nil {
			fields = append(fields, "error", err.Error())
		}

		logger.InfoContext(ctx, "grpc request completed", fields...)
		return resp, err
	}
}

func requestIDFromIncomingMetadata(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	values := md.Get("x-request-id")
	if len(values) == 0 {
		return "", false
	}

	requestID := strings.TrimSpace(values[0])
	if requestID == "" {
		return "", false
	}

	return requestID, true
}
