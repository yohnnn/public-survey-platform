package interceptors

import (
	"context"
	"strings"

	authv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/auth/v1"
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

func UnaryAuthInterceptor(authClient authv1.AuthServiceClient) grpc.UnaryServerInterceptor {
	publicMethods := map[string]struct{}{}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := publicMethods[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		token, err := extractBearerToken(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid bearer token")
		}

		validateResp, err := authClient.ValidateToken(ctx, &authv1.ValidateTokenRequest{AccessToken: token})
		if err != nil || !validateResp.GetValid() || strings.TrimSpace(validateResp.GetUserId()) == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, userIDCtxKey{}, validateResp.GetUserId())
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
