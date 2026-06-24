package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// NewRedis opens a Redis client and verifies connectivity.
func NewRedis(ctx context.Context, addr, password string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return client, nil
}
