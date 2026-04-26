package models

import "errors"

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrPollNotFound    = errors.New("poll not found")
	ErrTagNotFound     = errors.New("tag not found")
	ErrTagAlreadyExist = errors.New("tag already exists")
	ErrInvalidImageURL = errors.New("invalid image url")
	ErrImageTooLarge   = errors.New("image file is too large")
	ErrUnsupportedMime = errors.New("unsupported image content type")
	ErrImageUploadOff  = errors.New("image upload is disabled")
)
