package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Client wraps the reusable Redis driver.
type Client struct {
	client        *goredis.Client
	healthTimeout time.Duration
}

// New creates a Redis client without performing network I/O.
func New(config Config) *Client {
	return &Client{
		client: goredis.NewClient(&goredis.Options{
			Addr:        config.Address,
			Username:    config.Username,
			Password:    config.Password,
			DB:          config.Database,
			DialTimeout: config.DialTimeout,
		}),
		healthTimeout: config.HealthTimeout,
	}
}

// SDK returns the underlying Redis client.
func (client *Client) SDK() *goredis.Client {
	return client.client
}

// Ping verifies the Redis connection.
func (client *Client) Ping(ctx context.Context) error {
	if client.healthTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.healthTimeout)
		defer cancel()
	}
	return client.client.Ping(ctx).Err()
}

// Close closes the Redis client.
func (client *Client) Close() error {
	return client.client.Close()
}
