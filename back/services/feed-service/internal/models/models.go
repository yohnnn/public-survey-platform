package models

import "time"

type FeedItem struct {
	ID         string
	CreatorID  string
	Author     FeedAuthor
	Question   string
	ImageURL   string
	TotalVotes int64
	CreatedAt  time.Time
	Options    []FeedItemOption
	Tags       []string
}

type FeedItemOption struct {
	ID         string
	Text       string
	VotesCount int64
	Position   int32
}

type FeedAuthor struct {
	ID       string
	Nickname string
}
