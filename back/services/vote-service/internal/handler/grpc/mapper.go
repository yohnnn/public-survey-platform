package grpc

import (
	"time"

	votev1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/vote/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapVoteResponse(pollID string, optionIDs []string, votedAt time.Time) *votev1.VoteResponse {
	return &votev1.VoteResponse{
		PollId:    pollID,
		OptionIds: optionIDs,
		VotedAt:   timestamppb.New(votedAt),
	}
}

func mapGetUserVoteResponse(pollID string, hasVoted bool, optionIDs []string, votedAt *time.Time) *votev1.GetUserVoteResponse {
	resp := &votev1.GetUserVoteResponse{
		PollId:    pollID,
		HasVoted:  hasVoted,
		OptionIds: optionIDs,
	}
	if votedAt != nil {
		resp.VotedAt = timestamppb.New(votedAt.UTC())
	}
	return resp
}
