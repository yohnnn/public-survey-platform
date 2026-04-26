package grpc

import (
	analyticsv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/analytics/v1"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/models"
)

func mapPollAnalytics(item models.PollAnalytics) *analyticsv1.GetPollAnalyticsResponse {
	return &analyticsv1.GetPollAnalyticsResponse{
		PollId:     item.PollID,
		TotalVotes: item.TotalVotes,
		Options:    mapOptionStats(item.Options),
		Countries:  mapCountryStats(item.Countries),
		Gender:     mapGenderStats(item.Gender),
		Age:        mapAgeStats(item.Age),
	}
}

func mapOptionStats(items []models.OptionStat) []*analyticsv1.OptionStat {
	out := make([]*analyticsv1.OptionStat, 0, len(items))
	for _, item := range items {
		out = append(out, &analyticsv1.OptionStat{
			OptionId: item.OptionID,
			Votes:    item.Votes,
		})
	}
	return out
}

func mapCountryStats(items []models.CountryStat) []*analyticsv1.CountryStat {
	out := make([]*analyticsv1.CountryStat, 0, len(items))
	for _, item := range items {
		out = append(out, &analyticsv1.CountryStat{
			Country: item.Country,
			Votes:   item.Votes,
		})
	}
	return out
}

func mapGenderStats(items []models.GenderStat) []*analyticsv1.GenderStat {
	out := make([]*analyticsv1.GenderStat, 0, len(items))
	for _, item := range items {
		out = append(out, &analyticsv1.GenderStat{
			Gender: item.Gender,
			Votes:  item.Votes,
		})
	}
	return out
}

func mapAgeStats(items []models.AgeStat) []*analyticsv1.AgeStat {
	out := make([]*analyticsv1.AgeStat, 0, len(items))
	for _, item := range items {
		out = append(out, &analyticsv1.AgeStat{
			AgeRange: item.AgeRange,
			Votes:    item.Votes,
		})
	}
	return out
}
