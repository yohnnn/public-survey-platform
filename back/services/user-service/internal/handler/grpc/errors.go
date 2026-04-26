package grpc

import (
	"github.com/yohnnn/public-survey-platform/back/pkg/apperrors"
	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/models"
	"google.golang.org/grpc/codes"
)

var grpcErrorRules = []apperrors.GRPCRule{
	{Target: models.ErrInvalidArgument, Code: codes.InvalidArgument},
	{Target: models.ErrEmailAlreadyExists, Code: codes.AlreadyExists},
	{Target: models.ErrNicknameAlreadyExists, Code: codes.AlreadyExists},
	{Target: models.ErrCannotFollowSelf, Code: codes.InvalidArgument},
	{Target: models.ErrInvalidCredentials, Code: codes.Unauthenticated},
	{Target: models.ErrInvalidToken, Code: codes.Unauthenticated},
	{Target: models.ErrUnauthorized, Code: codes.Unauthenticated},
	{Target: models.ErrUserNotFound, Code: codes.NotFound},
	{Target: models.ErrSessionNotFound, Code: codes.NotFound},
	{Target: models.ErrSessionExpired, Code: codes.Unauthenticated},
	{Target: models.ErrSessionRevoked, Code: codes.Unauthenticated},
}

func toStatusError(err error) error {
	return apperrors.ToGRPC(err, grpcErrorRules...)
}
