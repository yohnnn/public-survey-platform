package kafka

import "time"

type pollCreatedPayload struct {
	EventID   string              `json:"event_id"`
	PollID    string              `json:"poll_id"`
	CreatorID string              `json:"creator_id"`
	Question  string              `json:"question"`
	Options   []pollCreatedOption `json:"options"`
	Tags      []string            `json:"tags"`
	CreatedAt time.Time           `json:"created_at"`
}

type pollUpdatedPayload struct {
	EventID   string    `json:"event_id"`
	PollID    string    `json:"poll_id"`
	Question  string    `json:"question"`
	Tags      []string  `json:"tags"`
	UpdatedAt time.Time `json:"updated_at"`
}

type pollDeletedPayload struct {
	EventID   string    `json:"event_id"`
	PollID    string    `json:"poll_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

type pollCreatedOption struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Position int32  `json:"position"`
}

type voteCastPayload struct {
	EventID   string   `json:"event_id"`
	PollID    string   `json:"poll_id"`
	OptionIDs []string `json:"option_ids"`
}

type voteRemovedPayload struct {
	EventID   string   `json:"event_id"`
	PollID    string   `json:"poll_id"`
	OptionIDs []string `json:"option_ids"`
}
