package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/inactivity"
)

// Store persists inactivity registry entries.
type Store struct{ pool *pgxpool.Pool }

// NewStore creates the PostgreSQL inactivity registry store.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// List returns every registry entry ordered by normalized name.
func (store *Store) List(ctx context.Context) ([]inactivity.Entry, error) {
	rows, err := store.pool.Query(ctx, `SELECT display_name, added_by, added_at
		FROM inactivity_dismissals ORDER BY normalized_name`)
	if err != nil {
		return nil, fmt.Errorf("list inactivity dismissals: %w", err)
	}
	defer rows.Close()
	entries := make([]inactivity.Entry, 0)
	for rows.Next() {
		var entry inactivity.Entry
		if err := rows.Scan(&entry.Name, &entry.AddedBy, &entry.AddedAt); err != nil {
			return nil, fmt.Errorf("scan inactivity dismissal: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inactivity dismissals: %w", err)
	}
	return entries, nil
}

// Add inserts one unique registry entry.
func (store *Store) Add(ctx context.Context, normalized string, display string, actor string) (inactivity.Entry, error) {
	var entry inactivity.Entry
	err := store.pool.QueryRow(ctx, `INSERT INTO inactivity_dismissals (normalized_name, display_name, added_by)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING RETURNING display_name, added_by, added_at`,
		normalized, display, actor).Scan(&entry.Name, &entry.AddedBy, &entry.AddedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return inactivity.Entry{}, inactivity.ErrAlreadyExists
	}
	if err != nil {
		return inactivity.Entry{}, fmt.Errorf("add inactivity dismissal: %w", err)
	}
	return entry, nil
}

// Remove deletes one registry entry by normalized name.
func (store *Store) Remove(ctx context.Context, normalized string) error {
	result, err := store.pool.Exec(ctx, `DELETE FROM inactivity_dismissals WHERE normalized_name = $1`, normalized)
	if err != nil {
		return fmt.Errorf("remove inactivity dismissal: %w", err)
	}
	if result.RowsAffected() == 0 {
		return inactivity.ErrNotFound
	}
	return nil
}
