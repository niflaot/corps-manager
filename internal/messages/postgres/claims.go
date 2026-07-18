package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/pixelados-net/discord-bot/internal/messages"
)

// MarkDue schedules one managed message for immediate reconciliation.
func (store *Store) MarkDue(ctx context.Context, key string) error {
	result, err := store.pool.Exec(ctx, `UPDATE managed_messages SET state = 'pending', next_check_at = now()
		WHERE key = $1 AND state <> 'archived'`, key)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return messages.ErrNotFound
	}
	return nil
}

// ClaimDue leases one bounded due batch without holding a network transaction.
func (store *Store) ClaimDue(ctx context.Context, request messages.ClaimRequest) ([]messages.Record, error) {
	rows, err := store.pool.Query(ctx, `WITH due AS (
		SELECT id AS due_id FROM managed_messages
		WHERE state NOT IN ('archived', 'blocked') AND next_check_at <= $1 AND (lease_until IS NULL OR lease_until <= $1)
		ORDER BY next_check_at, id LIMIT $2 FOR UPDATE SKIP LOCKED
	) UPDATE managed_messages AS message SET state = 'repairing', lease_owner = $3,
		lease_until = $1 + make_interval(secs => $4) FROM due WHERE message.id = due.due_id RETURNING `+recordColumns,
		request.Now, request.Limit, request.Owner, int64(request.LeaseDuration/time.Second))
	if err != nil {
		return nil, fmt.Errorf("claim managed messages: %w", err)
	}
	defer rows.Close()
	records := make([]messages.Record, 0, request.Limit)
	for rows.Next() {
		record, scanErr := scanRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

// Complete records one verified healthy Discord message.
func (store *Store) Complete(ctx context.Context, completion messages.Completion) error {
	result, err := store.pool.Exec(ctx, `UPDATE managed_messages SET discord_message_id = $4, observed_hash = $5,
		state = 'healthy', failure_count = 0, last_checked_at = $6,
		last_repaired_at = CASE WHEN $7 THEN $6 ELSE last_repaired_at END, next_check_at = $8,
		lease_owner = NULL, lease_until = NULL, last_error = NULL
		WHERE id = $1 AND lease_owner = $2 AND revision = $3`, completion.ID, completion.Owner,
		completion.Revision, completion.DiscordMessageID, completion.ObservedHash, completion.CheckedAt,
		completion.Repaired, completion.NextCheckAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return messages.ErrConflict
	}
	return nil
}

// Release records one failed attempt and clears its lease.
func (store *Store) Release(ctx context.Context, release messages.Release) error {
	result, err := store.pool.Exec(ctx, `UPDATE managed_messages SET state = $4, failure_count = failure_count + 1,
		last_checked_at = $5, next_check_at = $6, lease_owner = NULL, lease_until = NULL, last_error = $7
		WHERE id = $1 AND lease_owner = $2 AND revision = $3`, release.ID, release.Owner, release.Revision,
		release.State, release.CheckedAt, release.NextCheckAt, release.Error)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return messages.ErrConflict
	}
	return nil
}
