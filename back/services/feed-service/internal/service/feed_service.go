package service

import (
	"context"
	"strings"

	userv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/user/v1"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository"
	"google.golang.org/grpc/metadata"
)

type feedService struct {
	feedRepo   repository.FeedRepository
	userClient FollowingReader
}

func NewFeedService(feedRepo repository.FeedRepository, userClient FollowingReader) FeedService {
	return &feedService{
		feedRepo:   feedRepo,
		userClient: userClient,
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

func (s *feedService) GetFollowingFeed(ctx context.Context, userID, cursor string, limit uint32) ([]models.FeedItem, string, bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, "", false, models.ErrUnauthorized
	}
	if s.userClient == nil {
		return nil, "", false, models.ErrUnauthorized
	}

	listLimit := int(limit)
	if listLimit <= 0 {
		listLimit = 20
	}
	if listLimit > 100 {
		listLimit = 100
	}

	followingIDs, err := s.listFollowingIDs(ctx)
	if err != nil {
		return nil, "", false, err
	}
	if len(followingIDs) == 0 {
		return []models.FeedItem{}, "", false, nil
	}

	filter := repository.FeedListFilter{
		CreatorIDs: followingIDs,
		Limit:      listLimit + 1,
	}

	if strings.TrimSpace(cursor) != "" {
		createdAt, cursorID, err := decodeCursor(cursor)
		if err != nil {
			return nil, "", false, models.ErrInvalidArgument
		}
		filter.CursorCreatedAt = &createdAt
		filter.CursorID = cursorID
	}

	items, err := s.feedRepo.GetFollowingFeed(ctx, filter)
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

func (s *feedService) listFollowingIDs(ctx context.Context) ([]string, error) {
	outCtx := ctx
	if inMD, ok := metadata.FromIncomingContext(ctx); ok {
		authVals := inMD.Get("authorization")
		if len(authVals) > 0 && strings.TrimSpace(authVals[0]) != "" {
			outCtx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", authVals[0]))
		}
	}

	resp, err := s.userClient.ListMyFollowing(outCtx, &userv1.ListMyFollowingRequest{})
	if err != nil {
		return nil, err
	}

	userIDs := resp.GetUserIds()
	out := make([]string, 0, len(userIDs))
	seen := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		out = append(out, userID)
	}
	return out, nil
}

func (s *feedService) enrichItems(ctx context.Context, items []models.FeedItem) ([]models.FeedItem, error) {
	if len(items) == 0 {
		return items, nil
	}

	ids := make([]string, 0, len(items))
	creatorIDs := make([]string, 0, len(items))
	creatorSeen := make(map[string]struct{}, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
		creatorID := strings.TrimSpace(item.CreatorID)
		if creatorID == "" {
			continue
		}
		if _, ok := creatorSeen[creatorID]; ok {
			continue
		}
		creatorSeen[creatorID] = struct{}{}
		creatorIDs = append(creatorIDs, creatorID)
	}

	optionsMap, err := s.feedRepo.GetOptionsByFeedItemIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	tagsMap, err := s.feedRepo.GetTagsByFeedItemIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	authorsMap, err := s.getAuthorsByIDs(ctx, creatorIDs)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].Options = optionsMap[items[i].ID]
		items[i].Tags = tagsMap[items[i].ID]
		if author, ok := authorsMap[items[i].CreatorID]; ok {
			items[i].Author = author
		} else {
			items[i].Author = models.FeedAuthor{ID: items[i].CreatorID}
		}
	}

	return items, nil
}

func (s *feedService) getAuthorsByIDs(ctx context.Context, userIDs []string) (map[string]models.FeedAuthor, error) {
	out := make(map[string]models.FeedAuthor, len(userIDs))
	if len(userIDs) == 0 || s.userClient == nil {
		return out, nil
	}

	resp, err := s.userClient.BatchGetUserSummaries(ctx, &userv1.BatchGetUserSummariesRequest{UserIds: userIDs})
	if err != nil {
		return nil, err
	}

	for _, item := range resp.GetItems() {
		userID := strings.TrimSpace(item.GetId())
		if userID == "" {
			continue
		}
		out[userID] = models.FeedAuthor{
			ID:       userID,
			Nickname: strings.TrimSpace(item.GetNickname()),
		}
	}

	return out, nil
}
