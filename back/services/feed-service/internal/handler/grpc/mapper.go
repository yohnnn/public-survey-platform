package grpc

import (
	commonv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/common/v1"
	feedv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/feed/v1"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapFeedItem(model models.FeedItem) *feedv1.FeedItem {
	return &feedv1.FeedItem{
		Id:         model.ID,
		Question:   model.Question,
		ImageUrl:   model.ImageURL,
		TotalVotes: model.TotalVotes,
		Options:    mapFeedOptions(model.Options),
		Tags:       model.Tags,
		CreatedAt:  timestamppb.New(model.CreatedAt),
		Author:     mapFeedAuthor(model.Author),
	}
}

func mapFeedItems(items []models.FeedItem) []*feedv1.FeedItem {
	out := make([]*feedv1.FeedItem, 0, len(items))
	for _, item := range items {
		out = append(out, mapFeedItem(item))
	}
	return out
}

func mapFeedOptions(items []models.FeedItemOption) []*feedv1.FeedOption {
	out := make([]*feedv1.FeedOption, 0, len(items))
	for _, item := range items {
		out = append(out, &feedv1.FeedOption{
			Id:         item.ID,
			Text:       item.Text,
			VotesCount: item.VotesCount,
		})
	}
	return out
}

func mapFeedAuthor(author models.FeedAuthor) *feedv1.FeedAuthor {
	return &feedv1.FeedAuthor{
		Id:       author.ID,
		Nickname: author.Nickname,
	}
}

func mapCursorPageMeta(nextCursor string, hasMore bool, limit uint32) *commonv1.CursorPageMeta {
	return &commonv1.CursorPageMeta{
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Limit:      limit,
	}
}
