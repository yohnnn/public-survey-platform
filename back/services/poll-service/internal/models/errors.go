package models

import "errors"

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrPollNotFound    = errors.New("poll not found")
	ErrTagNotFound     = errors.New("tag not found")
	ErrTagAlreadyExist = errors.New("tag already exists")
)
