package redis

import (
	"context"

	"go.uber.org/fx"
)

// Module provides the configured Redis client and lifecycle cleanup.
var Module = fx.Module("redis", fx.Provide(LoadConfig, provideClient))

func provideClient(lifecycle fx.Lifecycle, config Config) *Client {
	client := New(config)
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		return client.Close()
	}})
	return client
}
