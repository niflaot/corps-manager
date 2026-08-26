package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/announcements"
)

// Store persists announcement cooldowns.
type Store struct{ pool *pgxpool.Pool }

// NewStore creates the PostgreSQL announcement store.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Acquire atomically creates or replaces an expired cooldown.
func (store *Store) Acquire(ctx context.Context, key string, announcedAt time.Time,
	availableAt time.Time, actor string) (announcements.State, error) {
	var state announcements.State
	err := store.pool.QueryRow(ctx, `INSERT INTO announcement_cooldowns
		(cooldown_key, actor, announced_at, available_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (cooldown_key) DO UPDATE SET actor = EXCLUDED.actor,
			announced_at = EXCLUDED.announced_at, available_at = EXCLUDED.available_at
		WHERE announcement_cooldowns.available_at <= EXCLUDED.announced_at
		RETURNING cooldown_key, actor, announced_at, available_at`,
		key, actor, announcedAt, availableAt).
		Scan(&state.Key, &state.Actor, &state.AnnouncedAt, &state.AvailableAt)
	if errors.Is(err, pgx.ErrNoRows) {
		current, getErr := store.Get(ctx, key)
		if getErr != nil {
			return announcements.State{}, getErr
		}
		return announcements.State{}, &announcements.CooldownActiveError{State: current}
	}
	if err != nil {
		return announcements.State{}, fmt.Errorf("acquire announcement cooldown: %w", err)
	}
	return state, nil
}

// Get returns one persisted cooldown.
func (store *Store) Get(ctx context.Context, key string) (announcements.State, error) {
	var state announcements.State
	err := store.pool.QueryRow(ctx, `SELECT cooldown_key, actor, announced_at, available_at
		FROM announcement_cooldowns WHERE cooldown_key = $1`, key).
		Scan(&state.Key, &state.Actor, &state.AnnouncedAt, &state.AvailableAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return announcements.State{}, announcements.ErrNotFound
	}
	if err != nil {
		return announcements.State{}, fmt.Errorf("get announcement cooldown: %w", err)
	}
	return state, nil
}

// Release removes the exact acquisition after a failed publication.
func (store *Store) Release(ctx context.Context, key string, announcedAt time.Time) error {
	if _, err := store.pool.Exec(ctx, `DELETE FROM announcement_cooldowns
		WHERE cooldown_key = $1 AND announced_at = $2`, key, announcedAt); err != nil {
		return fmt.Errorf("release announcement cooldown: %w", err)
	}
	return nil
}

// Clear removes a cooldown regardless of its current state.
func (store *Store) Clear(ctx context.Context, key string) error {
	if _, err := store.pool.Exec(ctx, `DELETE FROM announcement_cooldowns WHERE cooldown_key = $1`, key); err != nil {
		return fmt.Errorf("clear announcement cooldown: %w", err)
	}
	return nil
}
