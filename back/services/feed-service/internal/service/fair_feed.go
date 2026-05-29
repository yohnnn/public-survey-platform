package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository"
)

func (s *feedService) getFairFeed(ctx context.Context, cursor string, limit uint32, tags []string) ([]models.FeedItem, string, bool, error) {
	cfg := s.ranking.normalized()
	listLimit := normalizedListLimit(limit)

	discoveryLimit := discoveryFetchLimit(listLimit, cfg)

	discoveryFilter := repository.FeedListFilter{
		Limit:          discoveryLimit,
		Tags:           normalizeTags(tags),
		ExposureMaxAge: cfg.ExposureMaxAge,
	}
	chronFilter := repository.FeedListFilter{
		Limit: listLimit + 1,
		Tags:  normalizeTags(tags),
	}

	if strings.TrimSpace(cursor) != "" {
		discoveryCursor, chronCursor, err := decodeFairCursor(cursor)
		if err != nil {
			return nil, "", false, models.ErrInvalidArgument
		}
		if discoveryCursor != nil {
			discoveryFilter.DiscoveryCursor = discoveryCursor
		}
		if chronCursor.createdAt != nil {
			chronFilter.CursorCreatedAt = chronCursor.createdAt
			chronFilter.CursorID = chronCursor.id
		}
	}

	discoveryItems, err := s.feedRepo.GetDiscoveryFeed(ctx, discoveryFilter)
	if err != nil {
		return nil, "", false, err
	}
	chronItems, err := s.feedRepo.GetFeed(ctx, chronFilter)
	if err != nil {
		return nil, "", false, err
	}

	merged := mergeFairFeedItems(discoveryItems, chronItems, listLimit, cfg)
	enriched, err := s.enrichItems(ctx, merged.items)
	if err != nil {
		return nil, "", false, err
	}

	nextCursor := ""
	if merged.hasMore && len(enriched) > 0 {
		nextCursor = encodeFairCursor(merged.nextDiscovery, merged.nextChron)
	}

	return enriched, nextCursor, merged.hasMore, nil
}

type fairMergeResult struct {
	items         []models.FeedItem
	hasMore       bool
	nextDiscovery *repository.DiscoveryCursor
	nextChron     *chronCursorParts
}

type chronCursorParts struct {
	createdAt *time.Time
	id        string
}

func discoveryFetchLimit(listLimit int, cfg FeedRankingConfig) int {
	ratioTotal := cfg.DiscoverySlotsPerPage + cfg.ChronologicalSlotsPerPage
	if ratioTotal <= 0 {
		ratioTotal = 10
	}
	need := (listLimit*cfg.DiscoverySlotsPerPage + ratioTotal - 1) / ratioTotal
	if need < 1 {
		need = 1
	}
	return need + 1
}

func mergeFairFeedItems(discovery, chron []models.FeedItem, listLimit int, cfg FeedRankingConfig) fairMergeResult {
	cfg = cfg.normalized()

	seen := make(map[string]struct{}, len(discovery)+len(chron))
	di, ci := 0, 0
	out := make([]models.FeedItem, 0, listLimit+1)

	appendUnique := func(item models.FeedItem) bool {
		if _, ok := seen[item.ID]; ok {
			return false
		}
		seen[item.ID] = struct{}{}
		out = append(out, item)
		return true
	}

	for len(out) < listLimit+1 {
		added := false
		for i := 0; i < cfg.DiscoverySlotsPerPage && len(out) < listLimit+1; i++ {
			for di < len(discovery) {
				candidate := discovery[di]
				di++
				if appendUnique(candidate) {
					added = true
					break
				}
			}
		}
		for i := 0; i < cfg.ChronologicalSlotsPerPage && len(out) < listLimit+1; i++ {
			for ci < len(chron) {
				candidate := chron[ci]
				ci++
				if appendUnique(candidate) {
					added = true
					break
				}
			}
		}
		if !added {
			break
		}
	}

	hasMore := len(out) > listLimit || di < len(discovery) || ci < len(chron)
	if len(out) > listLimit {
		out = out[:listLimit]
	}

	result := fairMergeResult{items: out, hasMore: hasMore}
	if di > 0 {
		result.nextDiscovery = discoveryCursorFromItem(discovery[di-1])
	}
	if ci > 0 {
		result.nextChron = chronCursorFromItem(chron[ci-1])
	}
	return result
}

func discoveryCursorFromItem(item models.FeedItem) *repository.DiscoveryCursor {
	return &repository.DiscoveryCursor{
		ImpressionCount: item.ImpressionCount,
		CreatedAt:       item.CreatedAt,
		ID:              item.ID,
	}
}

func chronCursorFromItem(item models.FeedItem) *chronCursorParts {
	t := item.CreatedAt
	return &chronCursorParts{createdAt: &t, id: item.ID}
}

func encodeFairCursor(discovery *repository.DiscoveryCursor, chron *chronCursorParts) string {
	discoveryPart := ""
	if discovery != nil {
		discoveryPart = fmt.Sprintf("%d|%d|%s",
			discovery.ImpressionCount,
			discovery.CreatedAt.UTC().UnixNano(),
			discovery.ID,
		)
	}
	chronPart := ""
	if chron != nil && chron.createdAt != nil {
		chronPart = fmt.Sprintf("%d|%s", chron.createdAt.UTC().UnixNano(), chron.id)
	}
	raw := discoveryPart + ";" + chronPart
	return "fair:" + base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeFairCursor(cursor string) (*repository.DiscoveryCursor, *chronCursorParts, error) {
	cursor = strings.TrimSpace(cursor)
	if !strings.HasPrefix(cursor, "fair:") {
		return nil, nil, fmt.Errorf("invalid fair cursor")
	}
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(cursor, "fair:"))
	if err != nil {
		return nil, nil, err
	}
	parts := strings.SplitN(string(b), ";", 2)
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("invalid fair cursor")
	}

	var discovery *repository.DiscoveryCursor
	if strings.TrimSpace(parts[0]) != "" {
		discoveryParts := strings.SplitN(parts[0], "|", 3)
		if len(discoveryParts) != 3 {
			return nil, nil, fmt.Errorf("invalid discovery cursor")
		}
		impressions, err := strconv.ParseInt(discoveryParts[0], 10, 64)
		if err != nil {
			return nil, nil, err
		}
		createdNano, err := strconv.ParseInt(discoveryParts[1], 10, 64)
		if err != nil {
			return nil, nil, err
		}
		id := strings.TrimSpace(discoveryParts[2])
		if id == "" {
			return nil, nil, fmt.Errorf("invalid discovery cursor")
		}
		discovery = &repository.DiscoveryCursor{
			ImpressionCount: impressions,
			CreatedAt:       time.Unix(0, createdNano).UTC(),
			ID:              id,
		}
	}

	var chron *chronCursorParts
	if strings.TrimSpace(parts[1]) != "" {
		chronParts := strings.SplitN(parts[1], "|", 2)
		if len(chronParts) != 2 {
			return nil, nil, fmt.Errorf("invalid chron cursor")
		}
		createdNano, err := strconv.ParseInt(chronParts[0], 10, 64)
		if err != nil {
			return nil, nil, err
		}
		id := strings.TrimSpace(chronParts[1])
		if id == "" {
			return nil, nil, fmt.Errorf("invalid chron cursor")
		}
		t := time.Unix(0, createdNano).UTC()
		chron = &chronCursorParts{createdAt: &t, id: id}
	}

	return discovery, chron, nil
}

func (s *feedService) RecordFeedImpressions(ctx context.Context, viewerKey string, feedItemIDs []string) (int32, error) {
	resolved, err := s.resolveViewerKey(ctx, viewerKey)
	if err != nil {
		return 0, err
	}
	recorded, err := s.feedRepo.RecordFeedImpressions(ctx, resolved, feedItemIDs)
	if err != nil {
		return 0, err
	}
	return int32(recorded), nil
}

func (s *feedService) resolveViewerKey(ctx context.Context, requested string) (string, error) {
	if userID, ok := grpcinterceptor.UserIDFromContext(ctx); ok && strings.TrimSpace(userID) != "" {
		return strings.TrimSpace(userID), nil
	}
	viewerKey := strings.TrimSpace(requested)
	if viewerKey == "" {
		return "", models.ErrInvalidArgument
	}
	if len(viewerKey) > 128 {
		return "", models.ErrInvalidArgument
	}
	return viewerKey, nil
}

func normalizedListLimit(limit uint32) int {
	listLimit := int(limit)
	if listLimit <= 0 {
		listLimit = 20
	}
	if listLimit > 100 {
		listLimit = 100
	}
	return listLimit
}
