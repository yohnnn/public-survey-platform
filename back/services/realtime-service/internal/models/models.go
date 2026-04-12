package models

import "time"

type PollUpdateEvent struct {
	Event      string    `json:"event"`
	PollID     string    `json:"poll_id"`
	OptionIDs  []string  `json:"option_ids"`
	Delta      int64     `json:"delta"`
	TotalVotes int64     `json:"total_votes"`
	Timestamp  time.Time `json:"timestamp"`
}
