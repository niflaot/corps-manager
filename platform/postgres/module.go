package postgres

import (
	"context"

	"go.uber.org/fx"
)

// Module provides the configured PostgreSQL pool and lifecycle cleanup.
var Module = fx.Module("postgres", fx.Provide(LoadConfig, providePool))

func providePool(lifecycle fx.Lifecycle, config Config) (*Pool, error) {
	pool, err := New(context.Background(), config)
	if err != nil {
		return nil, err
	}
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		pool.Close()
		return nil
	}})
	return pool, nil
}
