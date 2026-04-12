package models

type OptionStat struct {
	OptionID string
	Votes    int64
}

type CountryStat struct {
	Country string
	Votes   int64
}

type GenderStat struct {
	Gender string
	Votes  int64
}

type AgeStat struct {
	AgeRange string
	Votes    int64
}

type PollAnalytics struct {
	PollID     string
	TotalVotes int64
	Options    []OptionStat
	Countries  []CountryStat
	Gender     []GenderStat
	Age        []AgeStat
}
