package cache

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func GetJSON(ctx context.Context, store Store, key string, dest any) (bool, error) {
	if store == nil {
		return false, nil
	}

	raw, err := store.Get(ctx, key)
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return false, err
	}

	return true, nil
}

func SetJSON(ctx context.Context, store Store, key string, value any, ttl time.Duration) error {
	if store == nil {
		return nil
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return store.Set(ctx, key, raw, ttl)
}

func GetInt64(ctx context.Context, store Store, key string) (int64, bool, error) {
	if store == nil {
		return 0, false, nil
	}

	raw, err := store.Get(ctx, key)
	if err != nil {
		if IsNotFound(err) {
			return 0, false, nil
		}
		return 0, false, err
	}

	v, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return 0, false, err
	}

	return v, true, nil
}
