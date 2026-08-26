// Package postgres persists managed messages in PostgreSQL.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/niflaot/corps-manager/internal/messages"
)

const recordColumns = `id::text, key, guild_id, channel_id, coalesce(discord_message_id, ''), payload,
desired_hash, coalesce(observed_hash, ''), revision, state, failure_count, last_checked_at,
last_repaired_at, coalesce(last_error, ''), created_at, updated_at`

// Store is the PostgreSQL managed-message repository.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a PostgreSQL managed-message repository.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// GetByKey returns one managed message by logical key.
func (store *Store) GetByKey(ctx context.Context, key string) (messages.Record, error) {
	record, err := scanRecord(store.pool.QueryRow(ctx, `SELECT `+recordColumns+` FROM managed_messages WHERE key = $1`, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return messages.Record{}, messages.ErrNotFound
	}
	return record, err
}

// List returns one filtered managed-message page.
func (store *Store) List(ctx context.Context, query messages.ListQuery) (messages.Page, error) {
	where := "WHERE true"
	args := make([]any, 0, 5)
	if query.State != "" {
		args = append(args, query.State)
		where += fmt.Sprintf(" AND state = $%d", len(args))
	}
	if query.GuildID != "" {
		args = append(args, query.GuildID)
		where += fmt.Sprintf(" AND guild_id = $%d", len(args))
	}
	if query.ChannelID != "" {
		args = append(args, query.ChannelID)
		where += fmt.Sprintf(" AND channel_id = $%d", len(args))
	}
	var total int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM managed_messages `+where, args...).Scan(&total); err != nil {
		return messages.Page{}, fmt.Errorf("count managed messages: %w", err)
	}
	args = append(args, query.Limit, query.Offset)
	rows, err := store.pool.Query(ctx, `SELECT `+recordColumns+` FROM managed_messages `+where+
		` ORDER BY created_at, id LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return messages.Page{}, fmt.Errorf("list managed messages: %w", err)
	}
	defer rows.Close()
	page := messages.Page{Items: []messages.Record{}, Total: total, Limit: query.Limit, Offset: query.Offset}
	for rows.Next() {
		record, scanErr := scanRecord(rows)
		if scanErr != nil {
			return messages.Page{}, scanErr
		}
		page.Items = append(page.Items, record)
	}
	if err := rows.Err(); err != nil {
		return messages.Page{}, fmt.Errorf("iterate managed messages: %w", err)
	}
	return page, nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanRecord(row rowScanner) (messages.Record, error) {
	return scanRecordWithPrefix(row)
}

func scanRecordWithPrefix(row rowScanner, prefix ...any) (messages.Record, error) {
	var record messages.Record
	var payload []byte
	destinations := append(prefix,
		&record.ID, &record.Key, &record.GuildID, &record.ChannelID, &record.DiscordMessageID,
		&payload, &record.DesiredHash, &record.ObservedHash, &record.Revision, &record.State,
		&record.FailureCount, &record.LastCheckedAt, &record.LastRepairedAt, &record.LastError,
		&record.CreatedAt, &record.UpdatedAt,
	)
	if err := row.Scan(destinations...); err != nil {
		return messages.Record{}, err
	}
	if err := json.Unmarshal(payload, &record.Payload); err != nil {
		return messages.Record{}, fmt.Errorf("decode managed message payload: %w", err)
	}
	record.Payload = record.Payload.Normalize()
	return record, nil
}

func encodePayload(payload messages.Payload) ([]byte, string, error) {
	payload = payload.Normalize()
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("encode managed message payload: %w", err)
	}
	hash, err := payload.Hash()
	return encoded, hash, err
}
