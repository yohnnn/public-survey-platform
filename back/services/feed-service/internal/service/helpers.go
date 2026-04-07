package service

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

func NewSystemClock() Clock {
	return systemClock{}
}

func encodeCursor(createdAt time.Time, id string) string {
	raw := fmt.Sprintf("%d|%s", createdAt.UTC().UnixNano(), id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cursor string) (time.Time, string, error) {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	nano, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, "", err
	}
	if strings.TrimSpace(parts[1]) == "" {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	return time.Unix(0, nano).UTC(), parts[1], nil
}

func encodeTrendingCursor(totalVotes int64, id string) string {
	raw := fmt.Sprintf("%d|%s", totalVotes, id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeTrendingCursor(cursor string) (int64, string, error) {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return 0, "", err
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid cursor")
	}
	votes, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", err
	}
	if strings.TrimSpace(parts[1]) == "" {
		return 0, "", fmt.Errorf("invalid cursor")
	}
	return votes, parts[1], nil
}

func normalizeTags(tags []string) []string {
	result := make([]string, 0, len(tags))
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
		result = append(result, v)
	}
	sort.Strings(result)
	return result
}
