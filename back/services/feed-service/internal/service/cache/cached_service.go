package servicecache

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	cachepkg "github.com/yohnnn/public-survey-platform/back/pkg/cache"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/service"
)

type Config struct {
	TTL time.Duration
}

func DefaultConfig() Config {
	return Config{TTL: 30 * time.Second}
}

type feedService struct {
	next  service.FeedService
	store cachepkg.Store
	ttl   time.Duration
}

func NewFeedService(next service.FeedService, store cachepkg.Store, cfg Config) service.FeedService {
	if next == nil || store == nil {
		return next
	}

	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 30 * time.Second
	}

	return &feedService{next: next, store: store, ttl: ttl}
}

func (s *feedService) GetFeed(ctx context.Context, cursor string, limit uint32, tags []string) ([]models.FeedItem, string, bool, error) {
	key := cacheKey("feed", strings.TrimSpace(cursor), strconv.Itoa(normalizedLimit(limit)), strings.Join(normalizedTags(tags), ","))
	var cached responseCache

	found, err := cachepkg.GetJSON(ctx, s.store, key, &cached)
	if err == nil && found {
		return cached.Items, cached.NextCursor, cached.HasMore, nil
	}

	items, nextCursor, hasMore, err := s.next.GetFeed(ctx, cursor, limit, tags)
	if err != nil {
		return nil, "", false, err
	}

	_ = cachepkg.SetJSON(ctx, s.store, key, responseCache{Items: items, NextCursor: nextCursor, HasMore: hasMore}, s.ttl)
	return items, nextCursor, hasMore, nil
}

func (s *feedService) GetTrending(ctx context.Context, cursor string, limit uint32) ([]models.FeedItem, string, bool, error) {
	key := cacheKey("trending", strings.TrimSpace(cursor), strconv.Itoa(normalizedLimit(limit)))
	var cached responseCache

	found, err := cachepkg.GetJSON(ctx, s.store, key, &cached)
	if err == nil && found {
		return cached.Items, cached.NextCursor, cached.HasMore, nil
	}

	items, nextCursor, hasMore, err := s.next.GetTrending(ctx, cursor, limit)
	if err != nil {
		return nil, "", false, err
	}

	_ = cachepkg.SetJSON(ctx, s.store, key, responseCache{Items: items, NextCursor: nextCursor, HasMore: hasMore}, s.ttl)
	return items, nextCursor, hasMore, nil
}

func (s *feedService) GetUserPolls(ctx context.Context, userID, cursor string, limit uint32) ([]models.FeedItem, string, bool, error) {
	key := cacheKey("user-polls", strings.TrimSpace(userID), strings.TrimSpace(cursor), strconv.Itoa(normalizedLimit(limit)))
	var cached responseCache

	found, err := cachepkg.GetJSON(ctx, s.store, key, &cached)
	if err == nil && found {
		return cached.Items, cached.NextCursor, cached.HasMore, nil
	}

	items, nextCursor, hasMore, err := s.next.GetUserPolls(ctx, userID, cursor, limit)
	if err != nil {
		return nil, "", false, err
	}

	_ = cachepkg.SetJSON(ctx, s.store, key, responseCache{Items: items, NextCursor: nextCursor, HasMore: hasMore}, s.ttl)
	return items, nextCursor, hasMore, nil
}

type responseCache struct {
	Items      []models.FeedItem `json:"items"`
	NextCursor string            `json:"next_cursor"`
	HasMore    bool              `json:"has_more"`
}

func cacheKey(kind string, parts ...string) string {
	return fmt.Sprintf("feed-service:%s:%s", kind, cachepkg.HashParts(parts...))
}

func normalizedLimit(limit uint32) int {
	v := int(limit)
	if v <= 0 {
		v = 20
	}
	if v > 100 {
		v = 100
	}
	return v
}

func normalizedTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, raw := range tags {
		v := strings.TrimSpace(strings.ToLower(raw))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
