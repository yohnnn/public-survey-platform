package grpc

import (
	"github.com/yohnnn/public-survey-platform/back/pkg/apperrors"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/models"
	"google.golang.org/grpc/codes"
)

var grpcErrorRules = []apperrors.GRPCRule{
	{Target: models.ErrInvalidArgument, Code: codes.InvalidArgument},
}

func toStatusError(err error) error {
	return apperrors.ToGRPC(err, grpcErrorRules...)
}
