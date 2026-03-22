package models

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Country      string
	Gender       string
	BirthYear    int32
	CreatedAt    time.Time
}

type RefreshSession struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	RevokedAt        *time.Time
}
