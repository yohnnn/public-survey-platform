package service

import (
	"context"

	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/models"
)

type RealtimeService interface {
	SubscribePollUpdates(ctx context.Context, pollID string) (<-chan models.PollUpdateEvent, func(), error)
	PublishPollUpdate(event models.PollUpdateEvent)
	WSHandshake(ctx context.Context) (string, error)
}
