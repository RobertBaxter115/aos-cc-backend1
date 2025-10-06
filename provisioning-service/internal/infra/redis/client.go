package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Client wraps the Redis client
type Client struct {
	rdb    *redis.Client
	logger *zap.Logger
}

// NewClient creates a new Redis client
func NewClient(addr, password string, db int, logger *zap.Logger) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	logger.Info("connected to redis",
		zap.String("addr", addr),
		zap.Int("db", db),
	)

	return &Client{
		rdb:    rdb,
		logger: logger,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// GetClient returns the underlying Redis client
func (c *Client) GetClient() *redis.Client {
	return c.rdb
}

// Subscribe subscribes to a Redis channel
func (c *Client) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channel)
}

// Publish publishes a message to a Redis channel
func (c *Client) Publish(ctx context.Context, channel, message string) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}
