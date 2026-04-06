package models

import "time"

type PollType int32

const (
	PollTypeUnspecified    PollType = 0
	PollTypeSingleChoice   PollType = 1
	PollTypeMultipleChoice PollType = 2
)

type Vote struct {
	UserID    string
	PollID    string
	OptionID  string
	CreatedAt time.Time
}
