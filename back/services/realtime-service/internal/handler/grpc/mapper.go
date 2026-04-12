package grpc

import (
	realtimev1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/realtime/v1"
	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapPollUpdateEvent(event models.PollUpdateEvent, optionID string) *realtimev1.StreamPollUpdatesResponse {
	return &realtimev1.StreamPollUpdatesResponse{
		Event:      event.Event,
		PollId:     event.PollID,
		OptionId:   optionID,
		TotalVotes: event.TotalVotes,
		Timestamp:  timestamppb.New(event.Timestamp),
	}
}
