package kafka

import "time"

type voteCastPayload struct {
	EventID   string    `json:"event_id"`
	UserID    string    `json:"user_id"`
	PollID    string    `json:"poll_id"`
	OptionIDs []string  `json:"option_ids"`
	Country   string    `json:"country,omitempty"`
	Gender    string    `json:"gender,omitempty"`
	BirthYear *int32    `json:"birth_year,omitempty"`
	VotedAt   time.Time `json:"voted_at"`
}

type voteRemovedPayload struct {
	EventID   string    `json:"event_id"`
	UserID    string    `json:"user_id"`
	PollID    string    `json:"poll_id"`
	OptionIDs []string  `json:"option_ids"`
	Country   string    `json:"country,omitempty"`
	Gender    string    `json:"gender,omitempty"`
	BirthYear *int32    `json:"birth_year,omitempty"`
	RemovedAt time.Time `json:"removed_at"`
}
