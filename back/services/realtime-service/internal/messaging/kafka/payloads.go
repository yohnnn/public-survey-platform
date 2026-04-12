package kafka

import "time"

type voteCastPayload struct {
	EventID   string    `json:"event_id"`
	PollID    string    `json:"poll_id"`
	OptionIDs []string  `json:"option_ids"`
	VotedAt   time.Time `json:"voted_at"`
}

type voteRemovedPayload struct {
	EventID   string    `json:"event_id"`
	PollID    string    `json:"poll_id"`
	OptionIDs []string  `json:"option_ids"`
	RemovedAt time.Time `json:"removed_at"`
}
