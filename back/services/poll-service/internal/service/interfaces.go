package service

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
)

type PollService interface {
	CreatePoll(ctx context.Context, userID, question string, pollType models.PollType, isAnonymous bool, endsAt *time.Time, options, tags []string) (models.Poll, error)
	GetPoll(ctx context.Context, id string) (models.Poll, error)
	ListPolls(ctx context.Context, cursor string, limit uint32, tags []string) ([]models.Poll, string, bool, error)
	UpdatePoll(ctx context.Context, userID, id string, question *string, isAnonymous *bool, endsAt *time.Time, tags []string, tagsProvided bool) (models.Poll, error)
	DeletePoll(ctx context.Context, userID, id string) error
	CreateTag(ctx context.Context, name string) (models.Tag, error)
	ListTags(ctx context.Context) ([]models.Tag, error)
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}
