package models

import "errors"

var (
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrFeedItemNotFound = errors.New("feed item not found")
)
