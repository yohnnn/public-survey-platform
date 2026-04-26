package models

import "time"

type User struct {
	ID           string
	Nickname     string
	Email        string
	PasswordHash string
	Country      string
	Gender       string
	BirthYear    int32
	CreatedAt    time.Time
}

type PublicProfile struct {
	ID             string
	Nickname       string
	FollowersCount int64
	FollowingCount int64
	IsFollowing    bool
}

type UserSummary struct {
	ID       string
	Nickname string
}

type RefreshSession struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	RevokedAt        *time.Time
}
