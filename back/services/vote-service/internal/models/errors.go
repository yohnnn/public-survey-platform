package models

import "errors"

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrPollNotFound    = errors.New("poll not found")
	ErrInvalidOption   = errors.New("invalid option")
)
