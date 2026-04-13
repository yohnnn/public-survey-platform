package grpc

import (
	"github.com/yohnnn/public-survey-platform/back/pkg/apperrors"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"google.golang.org/grpc/codes"
)

var grpcErrorRules = []apperrors.GRPCRule{
	{Target: models.ErrInvalidArgument, Code: codes.InvalidArgument},
	{Target: models.ErrFeedItemNotFound, Code: codes.NotFound},
}

func toStatusError(err error) error {
	return apperrors.ToGRPC(err, grpcErrorRules...)
}
