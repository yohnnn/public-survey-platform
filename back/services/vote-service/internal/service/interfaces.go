package service

import (
	"context"
	"time"
)

type VoteService interface {
	Vote(ctx context.Context, userID, pollID string, optionIDs []string) ([]string, time.Time, error)
	RemoveVote(ctx context.Context, userID, pollID string) error
	GetUserVote(ctx context.Context, userID, pollID string) (bool, []string, *time.Time, error)
}

type Clock interface {
	Now() time.Time
}
