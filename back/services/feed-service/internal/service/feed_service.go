package service

import (
	"context"
	"strings"

	userv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/user/v1"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository"
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

	followingIDs, err := ListFollowingIDs(ctx, s.userClient)
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

	type optResult struct {
		data map[string][]models.FeedItemOption
		err  error
	}
	type tagResult struct {
		data map[string][]string
		err  error
	}
	type authResult struct {
		data map[string]models.FeedAuthor
		err  error
	}

	optCh := make(chan optResult, 1)
	tagCh := make(chan tagResult, 1)
	authCh := make(chan authResult, 1)

	go func() {
		m, err := s.feedRepo.GetOptionsByFeedItemIDs(ctx, ids)
		optCh <- optResult{m, err}
	}()
	go func() {
		m, err := s.feedRepo.GetTagsByFeedItemIDs(ctx, ids)
		tagCh <- tagResult{m, err}
	}()
	go func() {
		m, err := s.getAuthorsByIDs(ctx, creatorIDs)
		authCh <- authResult{m, err}
	}()

	opt := <-optCh
	if opt.err != nil {
		return nil, opt.err
	}
	tag := <-tagCh
	if tag.err != nil {
		return nil, tag.err
	}
	auth := <-authCh
	if auth.err != nil {
		return nil, auth.err
	}

	for i := range items {
		items[i].Options = opt.data[items[i].ID]
		items[i].Tags = tag.data[items[i].ID]
		if author, ok := auth.data[items[i].CreatorID]; ok {
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
