package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pixelados-net/discord-bot/internal/messages"
)

// Create persists one idempotent managed-message definition.
func (store *Store) Create(ctx context.Context, definition messages.Definition, idempotency messages.Idempotency) (messages.MutationResult, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return messages.MutationResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if replay, found, err := reserveIdempotency(ctx, tx, idempotency); err != nil || found {
		return replay, err
	}
	payload, hash, err := encodePayload(definition.Payload)
	if err != nil {
		return messages.MutationResult{}, err
	}
	record, err := scanRecord(tx.QueryRow(ctx, `INSERT INTO managed_messages (key, guild_id, channel_id, payload, desired_hash)
		VALUES ($1, $2, $3, $4, $5) RETURNING `+recordColumns,
		definition.Key, definition.GuildID, definition.ChannelID, payload, hash))
	if err != nil {
		return messages.MutationResult{}, mapMutationError(err)
	}
	if err := finishIdempotency(ctx, tx, idempotency.Key, record, 201); err != nil {
		return messages.MutationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return messages.MutationResult{}, err
	}
	return messages.MutationResult{Record: record}, nil
}

// Replace atomically replaces desired state at one revision.
func (store *Store) Replace(ctx context.Context, key string, revision uint64, definition messages.Definition, idempotency messages.Idempotency) (messages.MutationResult, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return messages.MutationResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if replay, found, err := reserveIdempotency(ctx, tx, idempotency); err != nil || found {
		return replay, err
	}
	current, err := scanRecord(tx.QueryRow(ctx, `SELECT `+recordColumns+` FROM managed_messages WHERE key = $1 FOR UPDATE`, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return messages.MutationResult{}, messages.ErrNotFound
	}
	if err != nil {
		return messages.MutationResult{}, err
	}
	if current.Revision != revision || current.State == messages.StateArchived {
		return messages.MutationResult{}, messages.ErrConflict
	}
	payload, hash, err := encodePayload(definition.Payload)
	if err != nil {
		return messages.MutationResult{}, err
	}
	changed := !reflect.DeepEqual(definition.Payload.Normalize(), current.Payload.Normalize()) ||
		definition.GuildID != current.GuildID || definition.ChannelID != current.ChannelID
	record := current
	if changed {
		record, err = scanRecord(tx.QueryRow(ctx, `UPDATE managed_messages SET guild_id = $2, channel_id = $3, payload = $4,
			desired_hash = $5, discord_message_id = CASE WHEN guild_id = $2 AND channel_id = $3 THEN discord_message_id ELSE NULL END,
			observed_hash = NULL, revision = revision + 1, state = 'pending', failure_count = 0,
			next_check_at = now(), lease_owner = NULL, lease_until = NULL, last_error = NULL, updated_at = now()
			WHERE key = $1 RETURNING `+recordColumns, key, definition.GuildID, definition.ChannelID, payload, hash))
		if err != nil {
			return messages.MutationResult{}, mapMutationError(err)
		}
	}
	if err := finishIdempotency(ctx, tx, idempotency.Key, record, 200); err != nil {
		return messages.MutationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return messages.MutationResult{}, err
	}
	return messages.MutationResult{Record: record}, nil
}

// Archive stops reconciling one managed message.
func (store *Store) Archive(ctx context.Context, key string, revision uint64, idempotency messages.Idempotency) (messages.MutationResult, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return messages.MutationResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if replay, found, err := reserveIdempotency(ctx, tx, idempotency); err != nil || found {
		return replay, err
	}
	record, err := scanRecord(tx.QueryRow(ctx, `UPDATE managed_messages SET state = 'archived', revision = revision + 1,
		lease_owner = NULL, lease_until = NULL, updated_at = now() WHERE key = $1 AND revision = $2 AND state <> 'archived'
		RETURNING `+recordColumns, key, revision))
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if getErr := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM managed_messages WHERE key = $1)`, key).Scan(&exists); getErr != nil {
			return messages.MutationResult{}, getErr
		}
		if !exists {
			return messages.MutationResult{}, messages.ErrNotFound
		}
		return messages.MutationResult{}, messages.ErrConflict
	}
	if err != nil {
		return messages.MutationResult{}, err
	}
	if err := finishIdempotency(ctx, tx, idempotency.Key, record, 200); err != nil {
		return messages.MutationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return messages.MutationResult{}, err
	}
	return messages.MutationResult{Record: record}, nil
}

func reserveIdempotency(ctx context.Context, tx pgx.Tx, request messages.Idempotency) (messages.MutationResult, bool, error) {
	if _, err := tx.Exec(ctx, `DELETE FROM message_idempotency WHERE idempotency_key = $1 AND expires_at <= now()`, request.Key); err != nil {
		return messages.MutationResult{}, false, err
	}
	var inserted string
	err := tx.QueryRow(ctx, `INSERT INTO message_idempotency (idempotency_key, operation, request_hash)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING RETURNING idempotency_key`, request.Key, request.Operation, request.RequestHash).Scan(&inserted)
	if err == nil {
		return messages.MutationResult{}, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return messages.MutationResult{}, false, err
	}
	var operation, hash string
	var response []byte
	if err := tx.QueryRow(ctx, `SELECT operation, request_hash, response FROM message_idempotency WHERE idempotency_key = $1`, request.Key).Scan(&operation, &hash, &response); err != nil {
		return messages.MutationResult{}, false, err
	}
	if operation != request.Operation || hash != request.RequestHash || len(response) == 0 {
		return messages.MutationResult{}, false, messages.ErrConflict
	}
	var record messages.Record
	if err := json.Unmarshal(response, &record); err != nil {
		return messages.MutationResult{}, false, fmt.Errorf("decode idempotency response: %w", err)
	}
	return messages.MutationResult{Record: record, Replay: true}, true, nil
}

func finishIdempotency(ctx context.Context, tx pgx.Tx, key string, record messages.Record, status int) error {
	response, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE message_idempotency SET status_code = $2, response = $3, updated_at = now() WHERE idempotency_key = $1`, key, status, response)
	return err
}

func mapMutationError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return messages.ErrConflict
	}
	return err
}
