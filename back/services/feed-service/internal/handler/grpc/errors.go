package grpc

import (
	"errors"

	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
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
	case errors.Is(err, models.ErrFeedItemNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
