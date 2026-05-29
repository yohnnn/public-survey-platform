package service

import (
	"strings"
	"time"
)

type FeedRankingConfig struct {
	ExposureTargetDefault     int
	ExposureMaxAge            time.Duration
	DiscoverySlotsPerPage     int
	ChronologicalSlotsPerPage int
}

func DefaultFeedRankingConfig() FeedRankingConfig {
	return FeedRankingConfig{
		ExposureTargetDefault:     100,
		ExposureMaxAge:            168 * time.Hour,
		DiscoverySlotsPerPage:     3,
		ChronologicalSlotsPerPage: 7,
	}
}

func (c FeedRankingConfig) normalized() FeedRankingConfig {
	out := c
	if out.ExposureTargetDefault <= 0 {
		out.ExposureTargetDefault = 100
	}
	if out.ExposureMaxAge <= 0 {
		out.ExposureMaxAge = 168 * time.Hour
	}
	if out.DiscoverySlotsPerPage <= 0 {
		out.DiscoverySlotsPerPage = 3
	}
	if out.ChronologicalSlotsPerPage <= 0 {
		out.ChronologicalSlotsPerPage = 7
	}
	return out
}

func isChronologicalSort(sort string) bool {
	switch stringsTrimLower(sort) {
	case "chronological", "recent", "time":
		return true
	default:
		return false
	}
}

func stringsTrimLower(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
