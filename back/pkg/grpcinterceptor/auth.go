package grpcinterceptor

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type userIDCtxKey struct{}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDCtxKey{}).(string)
	return v, ok
}

type TokenValidator func(ctx context.Context, token string) (userID string, err error)

func UnaryAuthInterceptor(validate TokenValidator, publicMethods map[string]struct{}) grpc.UnaryServerInterceptor {
	if publicMethods == nil {
		publicMethods = map[string]struct{}{}
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := publicMethods[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		token, err := extractBearerToken(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid bearer token")
		}

		userID, err := validate(ctx, token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, userIDCtxKey{}, userID)
		return handler(ctx, req)
	}
}

func extractBearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization")
	}

	token := parseBearer(values[0])
	if token == "" {
		return "", status.Error(codes.Unauthenticated, "invalid authorization")
	}

	return token, nil
}

func parseBearer(raw string) string {
	parts := strings.SplitN(strings.TrimSpace(raw), " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
