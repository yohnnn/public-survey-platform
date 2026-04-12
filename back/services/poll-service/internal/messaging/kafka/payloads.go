package kafka

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
