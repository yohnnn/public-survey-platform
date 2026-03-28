package models

import "time"

type PollType int32

const (
	PollTypeUnspecified    PollType = 0
	PollTypeSingleChoice   PollType = 1
	PollTypeMultipleChoice PollType = 2
)

type Poll struct {
	ID          string
	CreatorID   string
	Question    string
	Type        PollType
	IsAnonymous bool
	EndsAt      *time.Time
	CreatedAt   time.Time
	TotalVotes  int64
	Options     []PollOption
	Tags        []string
}

type PollOption struct {
	ID         string
	Text       string
	VotesCount int64
	Position   int32
}

type Tag struct {
	ID        string
	Name      string
	CreatedAt time.Time
}
