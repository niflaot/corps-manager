package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is a reusable PostgreSQL connection pool.
type Pool struct {
	pool          *pgxpool.Pool
	healthTimeout time.Duration
}

// New creates a PostgreSQL pool without requiring an immediate connection.
func New(ctx context.Context, config Config) (*Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(config.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}
	poolConfig.MaxConns = config.MaxConns
	poolConfig.MinConns = config.MinConns
	poolConfig.ConnConfig.ConnectTimeout = config.ConnectTimeout
	poolConfig.AfterConnect = func(ctx context.Context, connection *pgx.Conn) error {
		_, err := connection.Exec(ctx, "select set_config('statement_timeout', $1, false)", config.StatementTimeout.String())
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	return &Pool{pool: pool, healthTimeout: config.HealthTimeout}, nil
}

// DB returns the underlying pgx pool.
func (pool *Pool) DB() *pgxpool.Pool {
	return pool.pool
}

// Ping verifies the PostgreSQL connection.
func (pool *Pool) Ping(ctx context.Context) error {
	if pool.healthTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, pool.healthTimeout)
		defer cancel()
	}
	return pool.pool.Ping(ctx)
}

// Close closes every PostgreSQL connection.
func (pool *Pool) Close() {
	pool.pool.Close()
}
