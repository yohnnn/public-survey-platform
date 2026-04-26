package repository

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/pkg/outbox"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
)

type PollListFilter struct {
	CursorCreatedAt *time.Time
	CursorID        string
	Limit           int
	Tags            []string
}

type PollPatch struct {
	Question *string
	ImageURL *string
}

type PollRepository interface {
	Create(ctx context.Context, poll models.Poll, options []models.PollOption, tagIDs []string) error
	GetByID(ctx context.Context, id string) (models.Poll, error)
	List(ctx context.Context, filter PollListFilter) ([]models.Poll, error)
	UpdateByIDAndCreator(ctx context.Context, pollID, creatorID string, patch PollPatch) error
	DeleteByIDAndCreator(ctx context.Context, pollID, creatorID string) error
	IncrementOptionVotes(ctx context.Context, pollID, optionID string, delta int64) error
	UpdateTotalVotes(ctx context.Context, pollID string, delta int64) error
	MarkEventProcessed(ctx context.Context, eventID, topic string) (bool, error)
	ReplaceTags(ctx context.Context, pollID string, tagIDs []string) error
	GetOptionsByPollIDs(ctx context.Context, pollIDs []string) (map[string][]models.PollOption, error)
	GetTagsByPollIDs(ctx context.Context, pollIDs []string) (map[string][]string, error)
}

type TagRepository interface {
	Create(ctx context.Context, tag models.Tag) (models.Tag, error)
	List(ctx context.Context) ([]models.Tag, error)
	EnsureByNames(ctx context.Context, names []string) ([]models.Tag, error)
}

type OutboxRepository interface {
	Add(ctx context.Context, event outbox.Event) error
	ListUnpublished(ctx context.Context, limit int) ([]outbox.Event, error)
	MarkPublished(ctx context.Context, eventID string, publishedAt time.Time) error
	MarkFailed(ctx context.Context, eventID, reason string) error
}
