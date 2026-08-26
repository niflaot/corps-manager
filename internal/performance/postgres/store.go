// Package postgres persists business performance aggregates in PostgreSQL.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/performance"
)

// Store is the PostgreSQL performance repository.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a PostgreSQL performance repository.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Get returns one persisted business aggregate.
func (store *Store) Get(ctx context.Context, businessID int64) (performance.State, error) {
	var encoded []byte
	var revision uint64
	var updatedAt time.Time
	row := store.pool.QueryRow(ctx, `SELECT state, revision, updated_at
		FROM business_performance_state WHERE business_id = $1`, businessID)
	var state performance.State
	if err := row.Scan(&encoded, &revision, &updatedAt); errors.Is(err, pgx.ErrNoRows) {
		return performance.State{}, performance.ErrNotFound
	} else if err != nil {
		return performance.State{}, fmt.Errorf("read business performance: %w", err)
	}
	if err := json.Unmarshal(encoded, &state); err != nil {
		return performance.State{}, fmt.Errorf("decode business performance: %w", err)
	}
	state.Revision = revision
	state.UpdatedAt = updatedAt
	return state, nil
}

// Save creates or compare-and-swaps one business aggregate.
func (store *Store) Save(ctx context.Context, state performance.State, expectedRevision uint64) (performance.State, error) {
	state.Revision = 0
	state.UpdatedAt = time.Time{}
	encoded, err := json.Marshal(state)
	if err != nil {
		return performance.State{}, fmt.Errorf("encode business performance: %w", err)
	}
	var revision uint64
	if expectedRevision == 0 {
		err = store.pool.QueryRow(ctx, `INSERT INTO business_performance_state (business_id, state)
			VALUES ($1, $2) ON CONFLICT DO NOTHING RETURNING revision, updated_at`,
			state.BusinessID, encoded).Scan(&revision, &state.UpdatedAt)
	} else {
		err = store.pool.QueryRow(ctx, `UPDATE business_performance_state
			SET state = $2, revision = revision + 1, updated_at = now()
			WHERE business_id = $1 AND revision = $3 RETURNING revision, updated_at`,
			state.BusinessID, encoded, expectedRevision).Scan(&revision, &state.UpdatedAt)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return performance.State{}, performance.ErrConflict
	}
	if err != nil {
		return performance.State{}, fmt.Errorf("write business performance: %w", err)
	}
	state.Revision = revision
	return state, nil
}
