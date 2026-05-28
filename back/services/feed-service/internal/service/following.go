package service

import (
	"context"
	"strings"

	userv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/user/v1"
	"google.golang.org/grpc/metadata"
)

func ListFollowingIDs(ctx context.Context, userClient FollowingReader) ([]string, error) {
	if userClient == nil {
		return nil, nil
	}

	outCtx := ctx
	if inMD, ok := metadata.FromIncomingContext(ctx); ok {
		authVals := inMD.Get("authorization")
		if len(authVals) > 0 && strings.TrimSpace(authVals[0]) != "" {
			outCtx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", authVals[0]))
		}
	}

	resp, err := userClient.ListMyFollowing(outCtx, &userv1.ListMyFollowingRequest{})
	if err != nil {
		return nil, err
	}

	userIDs := resp.GetUserIds()
	out := make([]string, 0, len(userIDs))
	seen := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		out = append(out, userID)
	}
	return out, nil
}
