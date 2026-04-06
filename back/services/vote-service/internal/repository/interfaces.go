package repository

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/pkg/outbox"
)

type VoteRepository interface {
	ReplaceUserVote(ctx context.Context, userID, pollID string, optionIDs []string, createdAt time.Time) error
	DeleteUserVote(ctx context.Context, userID, pollID string) error
	GetUserVote(ctx context.Context, userID, pollID string) ([]string, *time.Time, error)
}

type OutboxRepository interface {
	Add(ctx context.Context, event outbox.Event) error
	ListUnpublished(ctx context.Context, limit int) ([]outbox.Event, error)
	MarkPublished(ctx context.Context, eventID string, publishedAt time.Time) error
	MarkFailed(ctx context.Context, eventID, reason string) error
}
