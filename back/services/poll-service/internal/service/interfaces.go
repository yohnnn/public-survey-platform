package service

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
)

type PollService interface {
	CreatePoll(ctx context.Context, userID, question string, pollType models.PollType, options, tags []string, imageURL string) (models.Poll, error)
	GetPoll(ctx context.Context, id string) (models.Poll, error)
	ListPolls(ctx context.Context, cursor string, limit uint32, tags []string) ([]models.Poll, string, bool, error)
	UpdatePoll(ctx context.Context, userID, id string, question *string, tags []string, tagsProvided bool, imageURL *string) (models.Poll, error)
	DeletePoll(ctx context.Context, userID, id string) error
	CreateTag(ctx context.Context, name string) (models.Tag, error)
	ListTags(ctx context.Context) ([]models.Tag, error)
	CreatePollImageUploadURL(ctx context.Context, userID, fileName, contentType string, sizeBytes int64) (models.PollImageUpload, error)
}

type PollImageUploader interface {
	CreatePollImageUploadURL(ctx context.Context, userID, fileName, contentType string, sizeBytes int64) (models.PollImageUpload, error)
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}
