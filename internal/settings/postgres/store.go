// Package postgres persists application settings in PostgreSQL.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pixelados-net/discord-bot/internal/settings"
)

const columns = `key, value, revision, created_at, updated_at`

// Store is the PostgreSQL settings repository.
type Store struct{ pool *pgxpool.Pool }

// NewStore creates a PostgreSQL settings repository.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Get returns one persisted setting.
func (store *Store) Get(ctx context.Context, key settings.Key) (settings.Record, error) {
	return scan(store.pool.QueryRow(ctx, `SELECT `+columns+` FROM settings WHERE key=$1`, key))
}

// List returns all persisted settings.
func (store *Store) List(ctx context.Context) ([]settings.Record, error) {
	rows, err := store.pool.Query(ctx, `SELECT `+columns+` FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []settings.Record{}
	for rows.Next() {
		record, scanErr := scan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

// Set creates or updates one setting with optimistic concurrency.
func (store *Store) Set(ctx context.Context, key settings.Key, value []byte, revision uint64) (settings.Record, error) {
	if revision == 0 {
		record, err := scan(store.pool.QueryRow(ctx, `INSERT INTO settings(key,value) VALUES($1,$2)
			ON CONFLICT(key) DO NOTHING RETURNING `+columns, key, value))
		if err == settings.ErrNotFound {
			return settings.Record{}, settings.ErrConflict
		}
		return record, err
	}
	record, err := scan(store.pool.QueryRow(ctx, `UPDATE settings SET value=$2,revision=revision+1,updated_at=now()
		WHERE key=$1 AND revision=$3 RETURNING `+columns, key, value, revision))
	if err == settings.ErrNotFound {
		return settings.Record{}, settings.ErrConflict
	}
	return record, err
}

// Reset deletes one persisted override with optional optimistic concurrency.
func (store *Store) Reset(ctx context.Context, key settings.Key, revision uint64) error {
	command, err := store.pool.Exec(ctx, `DELETE FROM settings WHERE key=$1 AND ($2::bigint=0 OR revision=$2)`, key, revision)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		if revision > 0 {
			return settings.ErrConflict
		}
		return settings.ErrNotFound
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (settings.Record, error) {
	var record settings.Record
	if err := row.Scan(&record.Key, &record.Value, &record.Revision, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return settings.Record{}, settings.ErrNotFound
		}
		return settings.Record{}, err
	}
	return record, nil
}
