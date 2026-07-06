package database

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewRedis creates a new Redis client
func NewRedis(cfg *config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}
