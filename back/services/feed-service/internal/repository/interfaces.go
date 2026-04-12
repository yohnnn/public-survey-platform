package repository

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
)

type FeedListFilter struct {
	CursorCreatedAt *time.Time
	CursorID        string
	CursorVotes     *int64
	CreatorID       string
	Limit           int
	Tags            []string
}

type FeedRepository interface {
	CreateFeedItem(ctx context.Context, item models.FeedItem, options []models.FeedItemOption, tags []string) error
	UpdateFeedItem(ctx context.Context, item models.FeedItem, tags []string) error
	DeleteFeedItem(ctx context.Context, feedItemID string) error
	IncrementOptionVotes(ctx context.Context, optionID string, delta int64) error
	UpdateTotalVotes(ctx context.Context, feedItemID string, delta int64) error
	MarkEventProcessed(ctx context.Context, eventID, topic string) (bool, error)
	GetFeed(ctx context.Context, filter FeedListFilter) ([]models.FeedItem, error)
	GetTrending(ctx context.Context, filter FeedListFilter) ([]models.FeedItem, error)
	GetUserPolls(ctx context.Context, filter FeedListFilter) ([]models.FeedItem, error)
	GetOptionsByFeedItemIDs(ctx context.Context, feedItemIDs []string) (map[string][]models.FeedItemOption, error)
	GetTagsByFeedItemIDs(ctx context.Context, feedItemIDs []string) (map[string][]string, error)
}
