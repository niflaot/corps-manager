// Package postgres persists runtime settings in PostgreSQL.
package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists runtime settings.
type Store struct{ pool *pgxpool.Pool }

// NewStore creates the PostgreSQL settings store.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// List returns every setting whose key starts with the prefix.
func (store *Store) List(ctx context.Context, prefix string) (map[string]string, error) {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(prefix)
	rows, err := store.pool.Query(ctx, `SELECT setting_key, setting_value FROM bot_settings WHERE setting_key LIKE $1 || '%'`, escaped)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		values[key] = value
	}
	return values, rows.Err()
}

// SetMany atomically upserts all given settings.
func (store *Store) SetMany(ctx context.Context, values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for key, value := range values {
		batch.Queue(`INSERT INTO bot_settings (setting_key, setting_value) VALUES ($1, $2)
 ON CONFLICT (setting_key) DO UPDATE SET setting_value = EXCLUDED.setting_value, updated_at = now()`, key, value)
	}
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	results := transaction.SendBatch(ctx, batch)
	for range values {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return err
		}
	}
	if err := results.Close(); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}
