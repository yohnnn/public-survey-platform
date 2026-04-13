package apperrors

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCRule describes how a domain error should be mapped to a gRPC status code.
type GRPCRule struct {
	Target  error
	Code    codes.Code
	Message string
}

func ToGRPC(err error, rules ...GRPCRule) error {
	if err == nil {
		return nil
	}

	for _, rule := range rules {
		if rule.Target == nil {
			continue
		}
		if !errors.Is(err, rule.Target) {
			continue
		}

		message := rule.Message
		if message == "" {
			message = err.Error()
		}
		return status.Error(rule.Code, message)
	}

	return status.Error(codes.Internal, "internal error")
}
