package grpc

import (
	"github.com/yohnnn/public-survey-platform/back/pkg/apperrors"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
	"google.golang.org/grpc/codes"
)

var grpcErrorRules = []apperrors.GRPCRule{
	{Target: models.ErrInvalidArgument, Code: codes.InvalidArgument},
	{Target: models.ErrUnauthorized, Code: codes.Unauthenticated},
	{Target: models.ErrForbidden, Code: codes.PermissionDenied},
	{Target: models.ErrPollNotFound, Code: codes.NotFound},
	{Target: models.ErrTagNotFound, Code: codes.NotFound},
	{Target: models.ErrTagAlreadyExist, Code: codes.AlreadyExists},
	{Target: models.ErrInvalidImageURL, Code: codes.InvalidArgument},
	{Target: models.ErrImageTooLarge, Code: codes.InvalidArgument},
	{Target: models.ErrUnsupportedMime, Code: codes.InvalidArgument},
	{Target: models.ErrImageUploadOff, Code: codes.FailedPrecondition},
}

func toStatusError(err error) error {
	return apperrors.ToGRPC(err, grpcErrorRules...)
}
