package service

import (
	"context"
	"strings"

	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository"
)

type feedService struct {
	feedRepo repository.FeedRepository
}

func NewFeedService(feedRepo repository.FeedRepository) FeedService {
	return &feedService{
		feedRepo: feedRepo,
	}
}

func (s *feedService) GetFeed(ctx context.Context, cursor string, limit uint32, tags []string) ([]models.FeedItem, string, bool, error) {
	listLimit := int(limit)
	if listLimit <= 0 {
		listLimit = 20
	}
	if listLimit > 100 {
		listLimit = 100
	}

	filter := repository.FeedListFilter{
		Limit: listLimit + 1,
		Tags:  normalizeTags(tags),
	}

	if strings.TrimSpace(cursor) != "" {
		createdAt, cursorID, err := decodeCursor(cursor)
		if err != nil {
			return nil, "", false, models.ErrInvalidArgument
		}
		filter.CursorCreatedAt = &createdAt
		filter.CursorID = cursorID
	}

	items, err := s.feedRepo.GetFeed(ctx, filter)
	if err != nil {
		return nil, "", false, err
	}

	hasMore := len(items) > listLimit
	if hasMore {
		items = items[:listLimit]
	}

	enriched, err := s.enrichItems(ctx, items)
	if err != nil {
		return nil, "", false, err
	}

	nextCursor := ""
	if hasMore && len(enriched) > 0 {
		last := enriched[len(enriched)-1]
		nextCursor = encodeCursor(last.CreatedAt, last.ID)
	}

	return enriched, nextCursor, hasMore, nil
}

func (s *feedService) GetTrending(ctx context.Context, cursor string, limit uint32) ([]models.FeedItem, string, bool, error) {
	listLimit := int(limit)
	if listLimit <= 0 {
		listLimit = 20
	}
	if listLimit > 100 {
		listLimit = 100
	}

	filter := repository.FeedListFilter{
		Limit: listLimit + 1,
	}

	if strings.TrimSpace(cursor) != "" {
		votes, cursorID, err := decodeTrendingCursor(cursor)
		if err != nil {
			return nil, "", false, models.ErrInvalidArgument
		}
		filter.CursorVotes = &votes
		filter.CursorID = cursorID
	}

	items, err := s.feedRepo.GetTrending(ctx, filter)
	if err != nil {
		return nil, "", false, err
	}

	hasMore := len(items) > listLimit
	if hasMore {
		items = items[:listLimit]
	}

	enriched, err := s.enrichItems(ctx, items)
	if err != nil {
		return nil, "", false, err
	}

	nextCursor := ""
	if hasMore && len(enriched) > 0 {
		last := enriched[len(enriched)-1]
		nextCursor = encodeTrendingCursor(last.TotalVotes, last.ID)
	}

	return enriched, nextCursor, hasMore, nil
}

func (s *feedService) GetUserPolls(ctx context.Context, userID, cursor string, limit uint32) ([]models.FeedItem, string, bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, "", false, models.ErrInvalidArgument
	}

	listLimit := int(limit)
	if listLimit <= 0 {
		listLimit = 20
	}
	if listLimit > 100 {
		listLimit = 100
	}

	filter := repository.FeedListFilter{
		CreatorID: userID,
		Limit:     listLimit + 1,
	}

	if strings.TrimSpace(cursor) != "" {
		createdAt, cursorID, err := decodeCursor(cursor)
		if err != nil {
			return nil, "", false, models.ErrInvalidArgument
		}
		filter.CursorCreatedAt = &createdAt
		filter.CursorID = cursorID
	}

	items, err := s.feedRepo.GetUserPolls(ctx, filter)
	if err != nil {
		return nil, "", false, err
	}

	hasMore := len(items) > listLimit
	if hasMore {
		items = items[:listLimit]
	}

	enriched, err := s.enrichItems(ctx, items)
	if err != nil {
		return nil, "", false, err
	}

	nextCursor := ""
	if hasMore && len(enriched) > 0 {
		last := enriched[len(enriched)-1]
		nextCursor = encodeCursor(last.CreatedAt, last.ID)
	}

	return enriched, nextCursor, hasMore, nil
}

func (s *feedService) enrichItems(ctx context.Context, items []models.FeedItem) ([]models.FeedItem, error) {
	if len(items) == 0 {
		return items, nil
	}

	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	optionsMap, err := s.feedRepo.GetOptionsByFeedItemIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	tagsMap, err := s.feedRepo.GetTagsByFeedItemIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].Options = optionsMap[items[i].ID]
		items[i].Tags = tagsMap[items[i].ID]
	}

	return items, nil
}
