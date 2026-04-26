package models

import "errors"

var (
	ErrInvalidArgument       = errors.New("invalid argument")
	ErrUserNotFound          = errors.New("user not found")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrNicknameAlreadyExists = errors.New("nickname already exists")
	ErrCannotFollowSelf      = errors.New("cannot follow self")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionRevoked        = errors.New("session revoked")
	ErrSessionExpired        = errors.New("session expired")
	ErrInvalidToken          = errors.New("invalid token")
	ErrUnauthorized          = errors.New("unauthorized")
)
