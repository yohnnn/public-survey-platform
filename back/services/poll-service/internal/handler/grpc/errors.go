package grpc

import (
	"errors"

	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toStatusError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, models.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, models.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, models.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, models.ErrPollNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, models.ErrTagNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, models.ErrTagAlreadyExist):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
