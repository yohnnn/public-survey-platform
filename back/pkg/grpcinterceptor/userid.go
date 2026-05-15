package grpcinterceptor

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryUserIDInterceptor(publicMethods map[string]struct{}) grpc.UnaryServerInterceptor {
	if publicMethods == nil {
		publicMethods = map[string]struct{}{}
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		userID, hasUserID := userIDFromMetadata(ctx)

		if _, ok := publicMethods[info.FullMethod]; ok {
			if hasUserID {
				ctx = context.WithValue(ctx, userIDCtxKey{}, userID)
			}
			return handler(ctx, req)
		}

		if !hasUserID {
			return nil, status.Error(codes.Unauthenticated, "missing user authentication")
		}

		ctx = context.WithValue(ctx, userIDCtxKey{}, userID)
		return handler(ctx, req)
	}
}

func userIDFromMetadata(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	values := md.Get("x-user-id")
	if len(values) == 0 {
		return "", false
	}
	userID := strings.TrimSpace(values[0])
	if userID == "" {
		return "", false
	}
	return userID, true
}
