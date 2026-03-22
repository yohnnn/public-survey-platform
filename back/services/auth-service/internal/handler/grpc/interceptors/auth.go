package interceptors

import (
	"context"
	"strings"

	authv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/auth/v1"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type userIDCtxKey struct{}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDCtxKey{}).(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

func UnaryAuthInterceptor(svc service.AuthService) grpc.UnaryServerInterceptor {
	publicMethods := map[string]struct{}{
		authv1.AuthService_Register_FullMethodName:      {},
		authv1.AuthService_Login_FullMethodName:         {},
		authv1.AuthService_RefreshToken_FullMethodName:  {},
		authv1.AuthService_ValidateToken_FullMethodName: {},
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := publicMethods[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		token, err := extractBearerToken(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid bearer token")
		}

		userID, err := svc.ValidateToken(ctx, token)
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
		return "", models.ErrUnauthorized
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", models.ErrUnauthorized
	}

	token := parseBearer(values[0])
	if token == "" {
		return "", models.ErrUnauthorized
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
