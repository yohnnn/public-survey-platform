package grpc

import (
	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapPoll(model models.Poll) *pollv1.Poll {
	out := &pollv1.Poll{
		Id:         model.ID,
		CreatorId:  model.CreatorID,
		Question:   model.Question,
		Type:       pollv1.PollType(model.Type),
		ImageUrl:   model.ImageURL,
		CreatedAt:  timestamppb.New(model.CreatedAt),
		TotalVotes: model.TotalVotes,
		Options:    mapOptions(model.Options),
		Tags:       model.Tags,
	}
	return out
}

func mapPolls(items []models.Poll) []*pollv1.Poll {
	out := make([]*pollv1.Poll, 0, len(items))
	for _, item := range items {
		out = append(out, mapPoll(item))
	}
	return out
}

func mapOptions(items []models.PollOption) []*pollv1.PollOption {
	out := make([]*pollv1.PollOption, 0, len(items))
	for _, item := range items {
		out = append(out, &pollv1.PollOption{
			Id:         item.ID,
			Text:       item.Text,
			VotesCount: item.VotesCount,
		})
	}
	return out
}

func mapTag(model models.Tag) *pollv1.Tag {
	return &pollv1.Tag{
		Id:        model.ID,
		Name:      model.Name,
		CreatedAt: timestamppb.New(model.CreatedAt),
	}
}

func mapTags(items []models.Tag) []*pollv1.Tag {
	out := make([]*pollv1.Tag, 0, len(items))
	for _, item := range items {
		out = append(out, mapTag(item))
	}
	return out
}
